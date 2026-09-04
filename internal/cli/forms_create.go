package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/PostMyForm/postmyform-cli/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type stringListFlag []string

func (v *stringListFlag) String() string {
	return fmt.Sprint([]string(*v))
}

func (v *stringListFlag) Set(value string) error {
	*v = append(*v, value)
	return nil
}

func runFormsCreate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("forms create", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {}

	name := flags.String("name", "", "form name")
	destinationEmail := flags.String("destination-email", "", "notification email")
	successRedirectURL := flags.String("success-redirect-url", "", "success redirect URL")
	jsonOutput := flags.Bool("json", false, "write JSON output")

	var allowedOrigins stringListFlag
	flags.Var(&allowedOrigins, "allowed-origin", "allowed browser origin")

	if err := flags.Parse(args); err != nil {
		return ExitUsage
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "forms create does not accept positional arguments")
		return ExitUsage
	}
	if *name == "" {
		fmt.Fprintln(stderr, "--name is required")
		return ExitUsage
	}
	if *destinationEmail == "" {
		fmt.Fprintln(stderr, "--destination-email is required")
		return ExitUsage
	}

	request := api.CreateFormRequest{
		Name:             *name,
		DestinationEmail: openapi_types.Email(*destinationEmail),
	}

	if len(allowedOrigins) > 0 {
		origins := []string(allowedOrigins)
		request.AllowedOrigins = &origins
	}
	if *successRedirectURL != "" {
		request.SuccessRedirectUrl.Set(*successRedirectURL)
	}

	if _, err := json.Marshal(request); err != nil {
		fmt.Fprintln(stderr, "invalid local request")
		return ExitUsage
	}

	client, code := configuredClient(stderr)
	if code != ExitSuccess {
		return code
	}

	form, err := client.CreateForm(context.Background(), request)
	if err != nil {
		return writeAPIError(stderr, err)
	}

	if *jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(form); err != nil {
			fmt.Fprintln(stderr, "failed to write JSON output")
			return ExitAPI
		}
		return ExitSuccess
	}

	fmt.Fprintf(stdout, "Created form: %s\n", form.Id.String())
	fmt.Fprintf(stdout, "Name: %s\n", form.Name)
	fmt.Fprintf(stdout, "Status: %s\n", form.Status)
	fmt.Fprintf(stdout, "Submission URL: %s\n", form.SubmissionUrl)

	return ExitSuccess
}
