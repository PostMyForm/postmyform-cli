package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/google/uuid"
)

func runFormsSnippet(args []string, stdout, stderr io.Writer) int {
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
			fmt.Fprintf(stderr, "unknown option for forms snippet: %s\n", arg)
			return ExitUsage
		}
	}

	client, code := configuredClient(stderr)
	if code != ExitSuccess {
		return code
	}

	snippet, err := client.GetFormSnippet(context.Background(), formID)
	if err != nil {
		return writeAPIError(stderr, err)
	}

	if jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)

		if err := encoder.Encode(struct {
			HTML string `json:"html"`
		}{
			HTML: snippet.Html,
		}); err != nil {
			fmt.Fprintln(stderr, "failed to write JSON output")
			return ExitAPI
		}

		return ExitSuccess
	}

	if _, err := fmt.Fprint(stdout, snippet.Html); err != nil {
		fmt.Fprintln(stderr, "failed to write snippet output")
		return ExitAPI
	}

	return ExitSuccess
}
