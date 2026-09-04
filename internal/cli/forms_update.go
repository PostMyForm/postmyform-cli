package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/PostMyForm/postmyform-cli/internal/api"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type optionalStringFlag struct {
	set   bool
	value string
}

func (v *optionalStringFlag) String() string {
	return v.value
}

func (v *optionalStringFlag) Set(value string) error {
	v.set = true
	v.value = value
	return nil
}

func runFormsUpdate(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "form ID is required")
		return ExitUsage
	}

	formID, err := uuid.Parse(args[0])
	if err != nil {
		fmt.Fprintln(stderr, "form ID must be a valid UUID")
		return ExitUsage
	}

	flags := flag.NewFlagSet("forms update", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {}

	var name optionalStringFlag
	var destinationEmail optionalStringFlag
	var spamHoneypotField optionalStringFlag
	var statusValue optionalStringFlag
	var successRedirectURL optionalStringFlag
	var allowedOrigins stringListFlag

	flags.Var(&name, "name", "form name")
	flags.Var(&destinationEmail, "destination-email", "notification email")
	flags.Var(&spamHoneypotField, "spam-honeypot-field", "honeypot field name")
	flags.Var(&statusValue, "status", "form status: active or paused")
	flags.Var(&successRedirectURL, "success-redirect-url", "success redirect URL")
	flags.Var(&allowedOrigins, "allowed-origin", "allowed browser origin")

	clearAllowedOrigins := flags.Bool(
		"clear-allowed-origins",
		false,
		"clear all allowed origins",
	)
	clearSuccessRedirect := flags.Bool(
		"clear-success-redirect-url",
		false,
		"clear the success redirect URL",
	)
	jsonOutput := flags.Bool("json", false, "write JSON output")

	if err := flags.Parse(args[1:]); err != nil {
		return ExitUsage
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "forms update does not accept extra positional arguments")
		return ExitUsage
	}
	if *clearAllowedOrigins && len(allowedOrigins) > 0 {
		fmt.Fprintln(
			stderr,
			"--allowed-origin and --clear-allowed-origins cannot be used together",
		)
		return ExitUsage
	}
	if *clearSuccessRedirect && successRedirectURL.set {
		fmt.Fprintln(
			stderr,
			"--success-redirect-url and --clear-success-redirect-url cannot be used together",
		)
		return ExitUsage
	}

	request := api.PatchFormRequest{}
	changed := false

	if name.set {
		request.Name = &name.value
		changed = true
	}
	if destinationEmail.set {
		email := openapi_types.Email(destinationEmail.value)
		request.DestinationEmail = &email
		changed = true
	}
	if spamHoneypotField.set {
		request.SpamHoneypotField = &spamHoneypotField.value
		changed = true
	}
	if statusValue.set {
		status := api.EditableFormStatus(statusValue.value)
		if !status.Valid() {
			fmt.Fprintln(stderr, "--status must be active or paused")
			return ExitUsage
		}
		request.Status = &status
		changed = true
	}
	if len(allowedOrigins) > 0 {
		origins := []string(allowedOrigins)
		request.AllowedOrigins = &origins
		changed = true
	}
	if *clearAllowedOrigins {
		origins := []string{}
		request.AllowedOrigins = &origins
		changed = true
	}
	if successRedirectURL.set {
		request.SuccessRedirectUrl.Set(successRedirectURL.value)
		changed = true
	}
	if *clearSuccessRedirect {
		request.SuccessRedirectUrl.SetNull()
		changed = true
	}

	if !changed {
		fmt.Fprintln(stderr, "at least one update option is required")
		return ExitUsage
	}

	if _, err := json.Marshal(request); err != nil {
		fmt.Fprintln(stderr, "invalid local request")
		return ExitUsage
	}

	client, code := configuredClient(stderr)
	if code != ExitSuccess {
		return code
	}

	receipt, err := client.UpdateForm(context.Background(), formID, request)
	if err != nil {
		return writeAPIError(stderr, err)
	}

	if *jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(receipt); err != nil {
			fmt.Fprintln(stderr, "failed to write JSON output")
			return ExitAPI
		}
		return ExitSuccess
	}

	fmt.Fprintf(stdout, "Updated form: %s\n", receipt.Id.String())
	fmt.Fprintf(stdout, "Status: %s\n", receipt.Status)
	fmt.Fprintf(stdout, "Updated at: %s\n", receipt.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	return ExitSuccess
}
