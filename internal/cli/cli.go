package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/PostMyForm/postmyform-cli/internal/api"
	"github.com/google/uuid"
)

const (
	ExitSuccess = 0
	ExitUsage   = 2
	ExitAuth    = 3
	ExitAPI     = 4
	ExitNetwork = 5
)

var Version = "dev"

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return ExitSuccess
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp(stdout)
		return ExitSuccess
	case "version", "-v", "--version":
		fmt.Fprintln(stdout, Version)
		return ExitSuccess
	case "forms":
		return runForms(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		fmt.Fprintln(stderr, "Run 'postmyform help' for usage.")
		return ExitUsage
	}
}

func runForms(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "forms command is required")
		return ExitUsage
	}

	switch args[0] {
	case "list":
		return runFormsList(args[1:], stdout, stderr)
	case "create":
		return runFormsCreate(args[1:], stdout, stderr)
	case "update":
		return runFormsUpdate(args[1:], stdout, stderr)
	case "fields":
		return runFormsFields(args[1:], stdout, stderr)
	case "snippet":
		return runFormsSnippet(args[1:], stdout, stderr)
	case "get":
		return runFormsGet(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown forms command: %s\n", args[0])
		return ExitUsage
	}
}

func runFormsList(args []string, stdout, stderr io.Writer) int {
	jsonOutput := false

	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		default:
			fmt.Fprintf(stderr, "unknown option for forms list: %s\n", arg)
			return ExitUsage
		}
	}

	client, code := configuredClient(stderr)
	if code != ExitSuccess {
		return code
	}

	forms, err := client.ListForms(context.Background())
	if err != nil {
		return writeAPIError(stderr, err)
	}

	if jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)

		if err := encoder.Encode(struct {
			Forms []api.Form `json:"forms"`
		}{
			Forms: forms,
		}); err != nil {
			fmt.Fprintln(stderr, "failed to write JSON output")
			return ExitAPI
		}

		return ExitSuccess
	}

	if len(forms) == 0 {
		fmt.Fprintln(stdout, "No forms found.")
		return ExitSuccess
	}

	for _, form := range forms {
		fmt.Fprintf(
			stdout,
			"%s\t%s\t%s\n",
			form.Id.String(),
			form.Status,
			form.Name,
		)
	}

	return ExitSuccess
}

func runFormsGet(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "form ID is required")
		return ExitUsage
	}

	formID, err := uuid.Parse(args[0])
	if err != nil {
		fmt.Fprintln(stderr, "form ID must be a valid UUID")
		return ExitUsage
	}

	jsonOutput := false
	for _, arg := range args[1:] {
		switch arg {
		case "--json":
			jsonOutput = true
		default:
			fmt.Fprintf(stderr, "unknown option for forms get: %s\n", arg)
			return ExitUsage
		}
	}

	client, code := configuredClient(stderr)
	if code != ExitSuccess {
		return code
	}

	form, err := client.GetForm(context.Background(), formID)
	if err != nil {
		return writeAPIError(stderr, err)
	}

	if jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)

		if err := encoder.Encode(form); err != nil {
			fmt.Fprintln(stderr, "failed to write JSON output")
			return ExitAPI
		}

		return ExitSuccess
	}

	fmt.Fprintf(stdout, "ID: %s\n", form.Id.String())
	fmt.Fprintf(stdout, "Name: %s\n", form.Name)
	fmt.Fprintf(stdout, "Status: %s\n", form.Status)
	fmt.Fprintf(stdout, "Destination: %s\n", form.DestinationEmail)
	fmt.Fprintf(stdout, "Submission URL: %s\n", form.SubmissionUrl)

	return ExitSuccess
}

func configuredClient(stderr io.Writer) (*api.Client, int) {
	token := os.Getenv("POSTMYFORM_API_TOKEN")
	baseURL := os.Getenv("POSTMYFORM_API_BASE_URL")

	client, err := api.NewClient(baseURL, token, 0)
	if err == nil {
		return client, ExitSuccess
	}

	if errors.Is(err, api.ErrMissingToken) {
		fmt.Fprintln(stderr, "POSTMYFORM_API_TOKEN is required")
		return nil, ExitAuth
	}

	fmt.Fprintln(stderr, "invalid PostMyForm API configuration")
	return nil, ExitUsage
}

func writeAPIError(stderr io.Writer, err error) int {
	if errors.Is(err, context.DeadlineExceeded) {
		fmt.Fprintln(stderr, "timeout")
		return ExitNetwork
	}

	var netError net.Error
	if errors.As(err, &netError) && netError.Timeout() {
		fmt.Fprintln(stderr, "timeout")
		return ExitNetwork
	}

	var apiError *api.APIError
	if errors.As(err, &apiError) {
		category := ""

		switch {
		case apiError.Code == api.ErrorCodeUnauthorized ||
			apiError.StatusCode == 401:
			fmt.Fprintln(stderr, "authentication failure")
			return ExitAuth
		case apiError.Code == api.ErrorCodeInsufficientScope ||
			apiError.StatusCode == 403:
			fmt.Fprintln(stderr, "authorization or scope failure")
			return ExitAuth
		case apiError.Code == api.ErrorCodeInvalidRequest ||
			apiError.Code == api.ErrorCodePayloadTooLarge ||
			apiError.Code == api.ErrorCodeUnsupportedMediaType ||
			apiError.StatusCode == 400 ||
			apiError.StatusCode == 413 ||
			apiError.StatusCode == 415 ||
			apiError.StatusCode == 422:
			category = "validation failure"
		case apiError.Code == api.ErrorCodeNotFound ||
			apiError.StatusCode == 404:
			category = "not found"
		case apiError.StatusCode == 409:
			category = "conflict"
		case apiError.Code == api.ErrorCodeRateLimited ||
			apiError.StatusCode == 429:
			category = "rate limited"
		case apiError.StatusCode >= 500:
			category = "PostMyForm API server failure"
		}

		if category != "" {
			if apiError.Message != "" {
				fmt.Fprintf(stderr, "%s: %s\n", category, apiError.Message)
			} else {
				fmt.Fprintf(stderr, "%s\n", category)
			}

			if apiError.StatusCode == 429 && apiError.RetryAfter != "" {
				fmt.Fprintf(stderr, "Retry-After: %s\n", apiError.RetryAfter)
			}

			return ExitAPI
		}

		if apiError.Message != "" {
			fmt.Fprintf(stderr, "%s\n", apiError.Message)
		} else {
			fmt.Fprintf(
				stderr,
				"PostMyForm API request failed with HTTP %d\n",
				apiError.StatusCode,
			)
		}

		return ExitAPI
	}

	var requestEncodingError *api.RequestEncodingError
	if errors.As(err, &requestEncodingError) {
		fmt.Fprintln(stderr, "invalid local request")
		return ExitUsage
	}

	var transportError *api.TransportError
	if errors.As(err, &transportError) {
		fmt.Fprintln(stderr, "network failure")
		return ExitNetwork
	}

	fmt.Fprintln(stderr, err)
	return ExitAPI
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, `PostMyForm CLI

Usage:
  postmyform forms <command> [options]
  postmyform version
  postmyform help

Form commands:
  list
  get
  create
  update
  fields get
  fields replace
  snippet`)
}
