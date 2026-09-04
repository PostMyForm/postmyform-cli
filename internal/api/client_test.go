package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClientRequiresToken(t *testing.T) {
	t.Parallel()

	_, err := NewClient("", "", 0)
	if !errors.Is(err, ErrMissingToken) {
		t.Fatalf("NewClient() error = %v, want %v", err, ErrMissingToken)
	}
}

func TestNewClientRejectsNonLoopbackHTTP(t *testing.T) {
	t.Parallel()

	_, err := NewClient("http://example.com/api/v1", "test-token", 0)
	if !errors.Is(err, ErrInvalidBaseURL) {
		t.Fatalf("NewClient() error = %v, want %v", err, ErrInvalidBaseURL)
	}
}

func TestListFormsSendsBearerToken(t *testing.T) {
	t.Parallel()

	const token = "sentinel-list-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+token {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		if got := r.URL.Path; got != "/forms" {
			t.Fatalf("path = %q, want /forms", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, token, 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	forms, err := client.ListForms(context.Background())
	if err != nil {
		t.Fatalf("ListForms() error = %v", err)
	}
	if len(forms) != 0 {
		t.Fatalf("ListForms() returned %d forms, want 0", len(forms))
	}
}

func TestCreateFormDoesNotRetryAndRedactsToken(t *testing.T) {
	t.Parallel()

	const token = "sentinel-create-token"

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(
			`{"error":{"code":"invalid_request","message":"failure ` + token + `"}}`,
		))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, token, 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.CreateForm(context.Background(), CreateFormRequest{Name: "Retry test", DestinationEmail: "retry@example.com"})
	if err == nil {
		t.Fatal("CreateForm() error = nil, want failure")
	}
	if requestCount != 1 {
		t.Fatalf("request count = %d, want 1; error = %v", requestCount, err)
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("CreateForm() error exposed API token: %q", err)
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("CreateForm() error = %q, want redaction marker", err)
	}
}

func TestListFormsTimeoutReturnsTransportError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.ListForms(context.Background())
	if err == nil {
		t.Fatal("ListForms returned nil error, want timeout")
	}

	var transportError *TransportError
	if !errors.As(err, &transportError) {
		t.Fatalf("error = %T %v, want *TransportError", err, err)
	}
}

func TestListFormsAcceptsJSONContentTypeWithParameters(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	forms, err := client.ListForms(context.Background())
	if err != nil {
		t.Fatalf("ListForms() error = %v", err)
	}
	if len(forms) != 0 {
		t.Fatalf("ListForms() returned %d forms, want 0", len(forms))
	}
}

func TestListFormsRejectsUnexpectedContentType(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListForms(context.Background())
	if !errors.Is(err, ErrUnexpectedContentType) {
		t.Fatalf("ListForms() error = %v, want %v", err, ErrUnexpectedContentType)
	}
}

func TestAPIErrorDoesNotExposeUnexpectedContentTypeBody(t *testing.T) {
	t.Parallel()

	const sentinel = "sentinel-secret-in-html"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("<html>" + sentinel + "</html>"))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListForms(context.Background())
	if err == nil {
		t.Fatal("ListForms() error = nil, want failure")
	}

	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("ListForms() error = %T %v, want *APIError", err, err)
	}
	if apiError.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("StatusCode = %d, want %d", apiError.StatusCode, http.StatusTooManyRequests)
	}
	if apiError.RetryAfter != "60" {
		t.Fatalf("RetryAfter = %q, want %q", apiError.RetryAfter, "60")
	}
	if strings.Contains(err.Error(), sentinel) {
		t.Fatalf("API error exposed unexpected response body: %q", err)
	}
}

func TestListFormsRejectsMissingRequiredData(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListForms(context.Background())
	if !errors.Is(err, ErrMalformedResponse) {
		t.Fatalf("ListForms() error = %v, want %v", err, ErrMalformedResponse)
	}
}

func TestListFormsRejectsMissingRequiredFormFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{}]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListForms(context.Background())
	if !errors.Is(err, ErrMalformedResponse) {
		t.Fatalf("ListForms() error = %v, want %v", err, ErrMalformedResponse)
	}
}

func TestListFormsRejectsOversizedResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(strings.Repeat("x", int(MaxResponseBodyBytes)+1)))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListForms(context.Background())
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("ListForms() error = %v, want %v", err, ErrResponseTooLarge)
	}
}

func TestListFormsInterruptedResponseReturnsTransportError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", "100")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListForms(context.Background())
	if err == nil {
		t.Fatal("ListForms() error = nil, want interrupted response failure")
	}

	var transportError *TransportError
	if !errors.As(err, &transportError) {
		t.Fatalf("ListForms() error = %T %v, want *TransportError", err, err)
	}
}

func TestUpdateFormDoesNotRetry(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{
			"error":{
				"code":"internal_error",
				"message":"controlled failure"
			}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	var formID FormId
	_, err = client.UpdateForm(context.Background(), formID, PatchFormRequest{})
	if err == nil {
		t.Fatal("UpdateForm() error = nil, want failure")
	}
	if requestCount != 1 {
		t.Fatalf("request count = %d, want 1", requestCount)
	}
}

func TestReplaceFormFieldsDoesNotRetry(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{
			"error":{
				"code":"internal_error",
				"message":"controlled failure"
			}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	var formID FormId
	_, err = client.ReplaceFormFields(
		context.Background(),
		formID,
		ReplaceFormFieldsRequest{Fields: []ReplaceFormField{}},
	)
	if err == nil {
		t.Fatal("ReplaceFormFields() error = nil, want failure")
	}
	if requestCount != 1 {
		t.Fatalf("request count = %d, want 1", requestCount)
	}
}

func TestAPIErrorRedactsAuthorizationAndRetryAfter(t *testing.T) {
	t.Parallel()

	const token = "777777"
	const authorization = "Bearer " + token

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", token)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = fmt.Fprintf(w, `{
			"error":{
				"code":"rate_limited",
				"message":"request failed with Authorization: %s"
			}
		}`, authorization)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, token, 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListForms(context.Background())
	if err == nil {
		t.Fatal("ListForms() error = nil, want failure")
	}

	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("ListForms() error = %T %v, want *APIError", err, err)
	}

	if strings.Contains(apiError.Error(), token) {
		t.Fatalf("API error exposed credential: %q", apiError.Error())
	}
	if strings.Contains(apiError.RetryAfter, token) {
		t.Fatalf("RetryAfter exposed credential: %q", apiError.RetryAfter)
	}
	if apiError.RetryAfter != "[REDACTED]" {
		t.Fatalf("RetryAfter = %q, want %q", apiError.RetryAfter, "[REDACTED]")
	}
}

func TestListFormsRejectsUntrustedTLSCertificate(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListForms(context.Background())
	if err == nil {
		t.Fatal("ListForms() error = nil, want TLS verification failure")
	}

	var transportError *TransportError
	if !errors.As(err, &transportError) {
		t.Fatalf("ListForms() error = %T %v, want *TransportError", err, err)
	}
}

func TestAPIErrorPreservesPublicAuthorizationMeaning(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		code       ErrorCode
	}{
		{
			name:       "invalid credential",
			statusCode: http.StatusUnauthorized,
			code:       ErrorCodeUnauthorized,
		},
		{
			name:       "insufficient scope",
			statusCode: http.StatusForbidden,
			code:       ErrorCodeInsufficientScope,
		},
		{
			name:       "inaccessible resource",
			statusCode: http.StatusNotFound,
			code:       ErrorCodeNotFound,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.statusCode)
				_, _ = fmt.Fprintf(
					w,
					`{"error":{"code":%q,"message":"controlled failure"}}`,
					test.code,
				)
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token", 0)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			_, err = client.ListForms(context.Background())
			if err == nil {
				t.Fatal("ListForms() error = nil, want failure")
			}

			var apiError *APIError
			if !errors.As(err, &apiError) {
				t.Fatalf("ListForms() error = %T %v, want *APIError", err, err)
			}
			if apiError.StatusCode != test.statusCode {
				t.Fatalf("StatusCode = %d, want %d", apiError.StatusCode, test.statusCode)
			}
			if apiError.Code != test.code {
				t.Fatalf("Code = %q, want %q", apiError.Code, test.code)
			}
		})
	}
}

func TestAPIErrorRejectsInvalidRetryAfter(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "not-a-valid-retry-value")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{
			"error":{
				"code":"rate_limited",
				"message":"Too many requests"
			}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListForms(context.Background())
	if err == nil {
		t.Fatal("ListForms() error = nil, want failure")
	}

	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("ListForms() error = %T %v, want *APIError", err, err)
	}
	if apiError.RetryAfter != "" {
		t.Fatalf("RetryAfter = %q, want empty for invalid value", apiError.RetryAfter)
	}
}
