package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/PostMyForm/postmyform-cli/internal/api"
	"github.com/google/uuid"
)

const maxStructuredInputBytes = int64(1 << 20)

var errStructuredInputTooLarge = errors.New("structured input exceeds size limit")

func runFormsFieldsReplace(args []string, stdout, stderr io.Writer) int {
	return runFormsFieldsReplaceWithStdin(args, os.Stdin, stdout, stderr)
}

func runFormsFieldsReplaceWithStdin(
	args []string,
	stdin io.Reader,
	stdout,
	stderr io.Writer,
) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "form ID is required")
		return ExitUsage
	}

	formID, err := uuid.Parse(args[0])
	if err != nil {
		fmt.Fprintln(stderr, "form ID must be a valid UUID")
		return ExitUsage
	}

	flags := flag.NewFlagSet("forms fields replace", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {}

	filePath := flags.String("file", "", "JSON request file, or - for stdin")
	jsonOutput := flags.Bool("json", false, "write JSON output")

	if err := flags.Parse(args[1:]); err != nil {
		return ExitUsage
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "forms fields replace does not accept extra positional arguments")
		return ExitUsage
	}
	if *filePath == "" {
		fmt.Fprintln(stderr, "--file is required")
		return ExitUsage
	}

	input, err := readStructuredInput(*filePath, stdin)
	if err != nil {
		if errors.Is(err, errStructuredInputTooLarge) {
			fmt.Fprintln(stderr, "structured input exceeds 1 MiB")
		} else {
			fmt.Fprintln(stderr, "unable to read structured input")
		}
		return ExitUsage
	}

	var request api.ReplaceFormFieldsRequest
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		fmt.Fprintln(stderr, "structured input must be a valid fields replacement request")
		return ExitUsage
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		fmt.Fprintln(stderr, "structured input must contain one JSON document")
		return ExitUsage
	}

	for index, field := range request.Fields {
		if !field.FieldType.Valid() {
			fmt.Fprintf(
				stderr,
				"field %d has invalid fieldType %q\n",
				index,
				field.FieldType,
			)
			return ExitUsage
		}
	}

	client, code := configuredClient(stderr)
	if code != ExitSuccess {
		return code
	}

	fields, err := client.ReplaceFormFields(context.Background(), formID, request)
	if err != nil {
		return writeAPIError(stderr, err)
	}

	if *jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)

		if err := encoder.Encode(struct {
			FormID string          `json:"formId"`
			Fields []api.FormField `json:"fields"`
		}{
			FormID: formID.String(),
			Fields: fields,
		}); err != nil {
			fmt.Fprintln(stderr, "failed to write JSON output")
			return ExitAPI
		}

		return ExitSuccess
	}

	fmt.Fprintf(
		stdout,
		"Replaced %d fields for form: %s\n",
		len(fields),
		formID.String(),
	)

	return ExitSuccess
}

func readStructuredInput(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		return readBoundedInput(stdin)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return readBoundedInput(file)
}

func readBoundedInput(reader io.Reader) ([]byte, error) {
	input, err := io.ReadAll(io.LimitReader(reader, maxStructuredInputBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(input)) > maxStructuredInputBytes {
		return nil, errStructuredInputTooLarge
	}

	return input, nil
}
