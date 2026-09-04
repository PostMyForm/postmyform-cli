# PostMyForm CLI

The official command-line interface for the PostMyForm public API.

PostMyForm CLI lets you manage forms, fields, and generated form snippets from a terminal or automation script.

The CLI uses only the public PostMyForm API. It does not access PostMyForm databases, private application code, or internal routes.

## Features

- List forms.
- Get one form.
- Create a form.
- Update selected form properties.
- Get the complete field configuration.
- Replace the complete field configuration.
- Get generated starter HTML.
- Human-readable output by default.
- Stable JSON output for scripts.
- Explicit exit codes.
- Token authentication through an environment variable.
- No automatic retries for mutations.
- Native binaries for Linux, macOS, and Windows.

## Install

Download releases from:

https://github.com/PostMyForm/postmyform-cli/releases

The current release provides these binaries:

| Platform | Architecture | File |
| --- | --- | --- |
| Linux | amd64 | `postmyform-linux-amd64` |
| Linux | arm64 | `postmyform-linux-arm64` |
| macOS | amd64 | `postmyform-darwin-amd64` |
| macOS | arm64 | `postmyform-darwin-arm64` |
| Windows | amd64 | `postmyform-windows-amd64.exe` |

Each release also includes:

- `SHA256SUMS`
- one CycloneDX JSON SBOM for each target

### Linux

Example for amd64:

```bash
gh release download v0.1.0 \
  --repo PostMyForm/postmyform-cli \
  --pattern postmyform-linux-amd64 \
  --pattern SHA256SUMS
```

Verify the checksum:

```bash
sha256sum -c --ignore-missing SHA256SUMS
```

Make the downloaded binary executable:

```bash
chmod +x postmyform-linux-amd64
```

Install it on your path:

```bash
sudo install -m 0755 postmyform-linux-amd64 /usr/local/bin/postmyform
```

Verify the installation:

```bash
postmyform version
```

### macOS

Download the binary that matches your Mac architecture and the release `SHA256SUMS` file.

Calculate its SHA-256 value:

```bash
shasum -a 256 postmyform-darwin-arm64
```

Compare the result with the matching entry in `SHA256SUMS`.

Make the binary executable:

```bash
chmod +x postmyform-darwin-arm64
```

Install it:

```bash
sudo install -m 0755 postmyform-darwin-arm64 /usr/local/bin/postmyform
```

Use `postmyform-darwin-amd64` instead on an Intel Mac.

### Windows

Download:

```text
postmyform-windows-amd64.exe
```

Verify its checksum in PowerShell:

```powershell
Get-FileHash .\postmyform-windows-amd64.exe -Algorithm SHA256
```

Compare the result with the matching entry in `SHA256SUMS`.

You can rename the executable to `postmyform.exe` and place it in a directory that is included in your `PATH`.

## Authentication

Create an API credential in PostMyForm and set it in the environment:

```bash
export POSTMYFORM_API_TOKEN='your-api-token'
```

The CLI does not provide a normal command-line flag for the API token. This helps keep credentials out of shell history and process argument lists.

Do not commit API tokens to source control.

## API endpoint

The CLI uses the production PostMyForm API by default:

```text
https://postmyform.com/api/v1
```

For an approved alternate environment, set:

```bash
export POSTMYFORM_API_BASE_URL='https://example.invalid/api/v1'
```

The CLI requires HTTPS for remote API endpoints.

Plain HTTP is accepted only for loopback addresses such as `localhost` during local testing.

## Usage

```text
postmyform forms <command> [options]
postmyform version
postmyform help
```

Form commands:

```text
list
get
create
update
fields get
fields replace
snippet
```

## List forms

Human-readable output:

```bash
postmyform forms list
```

JSON output:

```bash
postmyform forms list --json
```

JSON has this top-level shape:

```json
{
  "forms": []
}
```

## Get a form

```bash
postmyform forms get FORM_ID
```

JSON output:

```bash
postmyform forms get FORM_ID --json
```

`FORM_ID` must be a valid UUID.

## Create a form

Required options:

```text
--name
--destination-email
```

Optional options:

```text
--allowed-origin
--success-redirect-url
--json
```

Example:

```bash
postmyform forms create \
  --name "Contact" \
  --destination-email "contact@example.com" \
  --allowed-origin "https://example.com"
```

Add more than one allowed origin by repeating the option:

```bash
postmyform forms create \
  --name "Contact" \
  --destination-email "contact@example.com" \
  --allowed-origin "https://example.com" \
  --allowed-origin "https://www.example.com"
```

Configure a redirect after successful submission:

```bash
postmyform forms create \
  --name "Contact" \
  --destination-email "contact@example.com" \
  --success-redirect-url "https://example.com/thanks"
```

Use JSON output for automation:

```bash
postmyform forms create \
  --name "Contact" \
  --destination-email "contact@example.com" \
  --json
```

A successful human-readable mutation receipt includes the new form ID, name, status, and submission URL.

## Update a form

Syntax:

```bash
postmyform forms update FORM_ID [options]
```

Supported options:

```text
--name
--destination-email
--allowed-origin
--clear-allowed-origins
--success-redirect-url
--clear-success-redirect-url
--spam-honeypot-field
--status
--json
```

`--status` accepts:

```text
active
paused
```

Only properties supplied on the command line are sent in the PATCH request.

Example:

```bash
postmyform forms update FORM_ID \
  --name "Support Contact"
```

Replace the complete allowed-origin list:

```bash
postmyform forms update FORM_ID \
  --allowed-origin "https://example.com" \
  --allowed-origin "https://www.example.com"
```

Clear all allowed origins:

```bash
postmyform forms update FORM_ID \
  --clear-allowed-origins
```

Set a success redirect:

```bash
postmyform forms update FORM_ID \
  --success-redirect-url "https://example.com/thanks"
```

Clear the success redirect:

```bash
postmyform forms update FORM_ID \
  --clear-success-redirect-url
```

Clear operations are different from omitted properties.

You cannot combine `--allowed-origin` with `--clear-allowed-origins`.

You also cannot combine `--success-redirect-url` with `--clear-success-redirect-url`.

At least one update option is required.

## Get form fields

```bash
postmyform forms fields get FORM_ID
```

JSON output:

```bash
postmyform forms fields get FORM_ID --json
```

JSON has this top-level shape:

```json
{
  "fields": []
}
```

## Replace form fields

`fields replace` replaces the complete ordered field collection for a form. It does not append fields.

Use a JSON file:

```bash
postmyform forms fields replace FORM_ID \
  --file fields.json
```

Or read JSON from standard input:

```bash
cat fields.json | postmyform forms fields replace FORM_ID --file -
```

Use JSON output:

```bash
postmyform forms fields replace FORM_ID \
  --file fields.json \
  --json
```

Example input:

```json
{
  "fields": [
    {
      "name": "name",
      "label": "Name",
      "fieldType": "text",
      "required": true,
      "options": null
    },
    {
      "name": "email",
      "label": "Email",
      "fieldType": "email",
      "required": true,
      "options": null
    },
    {
      "name": "topic",
      "label": "Topic",
      "fieldType": "select",
      "required": false,
      "options": [
        "Sales",
        "Support",
        "Other"
      ]
    }
  ]
}
```

Supported field types are:

```text
text
email
textarea
select
checkbox
```

Structured input:

- must contain exactly one JSON document
- must not exceed 1 MiB
- rejects unknown JSON properties
- represents the complete replacement collection
- is validated before the API mutation is sent

A `select` field requires at least one option.

## Get a generated form snippet

Print generated HTML directly to standard output:

```bash
postmyform forms snippet FORM_ID
```

Write JSON instead:

```bash
postmyform forms snippet FORM_ID --json
```

JSON has this shape:

```json
{
  "html": "<form>...</form>"
}
```

The CLI prints the returned HTML. It does not execute, evaluate, or modify the snippet.

You can write it to a file:

```bash
postmyform forms snippet FORM_ID > form.html
```

## JSON output

Commands that support structured output use `--json`.

JSON data is written to standard output.

Errors are written to standard error.

This separation is intended for scripts and automation.

Example:

```bash
postmyform forms list --json > forms.json
```

With `jq`:

```bash
postmyform forms list --json | jq '.forms'
```

## Exit codes

| Code | Meaning |
| ---: | --- |
| `0` | Success |
| `2` | Invalid command, option, input, or local configuration |
| `3` | Authentication or authorization failure |
| `4` | PostMyForm API or response failure |
| `5` | Network or transport failure |

Scripts should test the exit code instead of parsing human-readable error messages.

## Rate limits

If the API returns HTTP `429` and provides a `Retry-After` value, the CLI writes that value to standard error.

The CLI does not automatically retry mutations.

This avoids repeating a create, update, or field replacement request when the first request might already have succeeded.

## Security behavior

The CLI:

- reads the API token from `POSTMYFORM_API_TOKEN`
- does not accept the token through a normal CLI argument
- does not intentionally print the token
- redacts the token if it appears in an API error message
- requires HTTPS for non-loopback API endpoints
- uses the operating system trust store
- applies bounded HTTP response sizes
- applies bounded request timeouts
- treats API responses as untrusted input
- does not execute shell commands from API responses
- does not execute generated HTML
- does not automatically retry mutations
- does not access the PostMyForm database
- does not use private PostMyForm application interfaces

## Development

Requirements:

- Go 1.27.1

Clone the repository:

```bash
git clone https://github.com/PostMyForm/postmyform-cli.git
cd postmyform-cli
```

Run tests:

```bash
go test ./...
```

Run the race detector:

```bash
go test -race ./...
```

Run static checks:

```bash
go vet ./...
```

Check formatting:

```bash
gofmt -l .
```

Build locally:

```bash
CGO_ENABLED=0 go build \
  -trimpath \
  -o postmyform \
  ./cmd/postmyform
```

A development build reports `dev` unless version metadata is injected at build time.

## API models

The repository contains a pinned copy of the public PostMyForm OpenAPI contract:

```text
api/openapi.json
```

Its expected SHA-256 is recorded in:

```text
api/openapi.sha256
```

Generated API models are created from the public contract.

The CLI does not depend on PostMyForm private application source code.

## Releases

Release tags use semantic version names such as `v0.1.0`.

A release tag must point to a commit that is already on `main`.

The release workflow:

1. validates the Go source
2. builds five native binaries
3. injects the release tag as the CLI version
4. generates a CycloneDX SBOM for each target
5. generates `SHA256SUMS`
6. publishes the files as a GitHub Release

Release targets:

```text
linux/amd64
linux/arm64
darwin/amd64
darwin/arm64
windows/amd64
```

Release builds use `CGO_ENABLED=0`, `-trimpath`, and `-buildvcs=false`.

## License

PostMyForm CLI is licensed under the Apache License 2.0.

See [LICENSE](LICENSE).
