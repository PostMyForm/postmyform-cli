package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBaseURL       = "https://postmyform.com/api/v1"
	DefaultTimeout       = 5 * time.Second
	MaxResponseBodyBytes = int64(1 << 20)
)

var (
	ErrMissingToken          = errors.New("POSTMYFORM_API_TOKEN is required")
	ErrInvalidBaseURL        = errors.New("invalid PostMyForm API base URL")
	ErrResponseTooLarge      = errors.New("PostMyForm API response exceeds size limit")
	ErrMalformedResponse     = errors.New("PostMyForm API returned a malformed response")
	ErrUnexpectedContentType = errors.New("PostMyForm API returned an unexpected content type")
)

type Client struct {
	baseURL    *url.URL
	token      string
	httpClient *http.Client
}

type APIError struct {
	StatusCode int
	Code       ErrorCode
	Message    string
	RetryAfter string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("PostMyForm API error (%d): %s", e.StatusCode, e.Message)
	}

	return fmt.Sprintf("PostMyForm API error (%d)", e.StatusCode)
}

type TransportError struct {
	err error
}

func (e *TransportError) Error() string {
	return "PostMyForm API transport failed"
}

func (e *TransportError) Unwrap() error {
	return e.err
}

func NewClient(baseURL, token string, timeout time.Duration) (*Client, error) {
	if token == "" {
		return nil, ErrMissingToken
	}

	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidBaseURL, err)
	}

	if parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, ErrInvalidBaseURL
	}

	switch parsed.Scheme {
	case "https":
	case "http":
		if !isLoopbackHost(parsed.Hostname()) {
			return nil, ErrInvalidBaseURL
		}
	default:
		return nil, ErrInvalidBaseURL
	}

	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")

	return &Client{
		baseURL: parsed,
		token:   token,
		httpClient: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (c *Client) do(
	ctx context.Context,
	method string,
	path string,
	body io.Reader,
	successStatus int,
	result any,
) error {
	requestURL := *c.baseURL
	requestURL.Path = strings.TrimRight(c.baseURL.Path, "/") + path

	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), body)
	if err != nil {
		return fmt.Errorf("create PostMyForm API request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &TransportError{err: err}
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBodyBytes+1))
	if err != nil {
		return &TransportError{err: err}
	}
	if int64(len(responseBody)) > MaxResponseBodyBytes {
		return ErrResponseTooLarge
	}

	if resp.StatusCode != successStatus {
		return c.decodeAPIError(resp, responseBody)
	}

	if result == nil {
		return nil
	}

	if len(responseBody) == 0 {
		return ErrMalformedResponse
	}

	if !isJSONContentType(resp.Header.Get("Content-Type")) {
		return ErrUnexpectedContentType
	}

	if err := validateResponseShape(responseBody, result); err != nil {
		return err
	}

	if err := json.Unmarshal(responseBody, result); err != nil {
		return fmt.Errorf("%w: %v", ErrMalformedResponse, err)
	}

	return nil
}

func (c *Client) decodeAPIError(resp *http.Response, body []byte) error {
	apiError := &APIError{
		StatusCode: resp.StatusCode,
		RetryAfter: c.sanitizeRetryAfter(resp.Header.Get("Retry-After")),
	}

	var errorResponse ErrorResponse
	if len(body) > 0 && isJSONContentType(resp.Header.Get("Content-Type")) && json.Unmarshal(body, &errorResponse) == nil {
		apiError.Code = errorResponse.Error.Code
		apiError.Message = c.redact(errorResponse.Error.Message)
	}

	return apiError
}

func (c *Client) sanitizeRetryAfter(value string) string {
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 1 {
		return ""
	}

	return c.redact(value)
}

func (c *Client) redact(value string) string {
	if c.token == "" {
		return value
	}

	return strings.ReplaceAll(value, c.token, "[REDACTED]")
}

func isJSONContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return false
	}

	return mediaType == "application/json"
}
