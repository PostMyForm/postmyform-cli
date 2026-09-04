package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/PostMyForm/postmyform-cli/internal/api"
	"github.com/google/uuid"
)

func runFormsFields(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "fields command is required")
		return ExitUsage
	}

	switch args[0] {
	case "get":
		return runFormsFieldsGet(args[1:], stdout, stderr)
	case "replace":
		return runFormsFieldsReplace(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown fields command: %s\n", args[0])
		return ExitUsage
	}
}

func runFormsFieldsGet(args []string, stdout, stderr io.Writer) int {
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
			fmt.Fprintf(stderr, "unknown option for forms fields get: %s\n", arg)
			return ExitUsage
		}
	}

	client, code := configuredClient(stderr)
	if code != ExitSuccess {
		return code
	}

	fields, err := client.ListFormFields(context.Background(), formID)
	if err != nil {
		return writeAPIError(stderr, err)
	}

	if jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)

		if err := encoder.Encode(struct {
			Fields []api.FormField `json:"fields"`
		}{
			Fields: fields,
		}); err != nil {
			fmt.Fprintln(stderr, "failed to write JSON output")
			return ExitAPI
		}

		return ExitSuccess
	}

	if len(fields) == 0 {
		fmt.Fprintln(stdout, "No fields found.")
		return ExitSuccess
	}

	for _, field := range fields {
		fmt.Fprintf(
			stdout,
			"%s\t%s\t%s\t%t\n",
			field.Name,
			field.FieldType,
			field.Label,
			field.Required,
		)
	}

	return ExitSuccess
}
