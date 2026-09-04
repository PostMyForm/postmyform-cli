# PostMyForm CLI contract and security validation

Date: 2026-09-04

## Scope

This document records the CLI portion of the PostMyForm developer-tooling
contract, security, compatibility, and staging validation.

The validation covers the public PostMyForm API boundary only.

It does not use:

- private PostMyForm source
- private PostMyForm types
- private API routes
- direct database access
- operator interfaces
- production customer data

Cross-product CLI/MCP compatibility and MCP protocol validation remain pending
until the PostMyForm MCP implementation is available.

## Tested source

Repository:

```text
PostMyForm/postmyform-cli
```

Validation branch:

```text
chore/295-cli-contract-security
```

Baseline revision:

```text
b339de8b646a8793b9ebdefe8111b83146253c14
```

The final validated pull-request revision and CI run will be recorded during
merge closeout.

## Public API contract

Authoritative contract:

```text
https://postmyform.com/openapi.json
```

Contract properties:

```text
OpenAPI version: 3.1.2
PostMyForm API version: 0.6.0
```

Supported snapshot:

```text
api/openapi.json
```

Supported snapshot SHA-256:

```text
40963e80cc46679ee4c78440c80d1e94c57083d3b3be6a59a6b9438c7f0c756a
```

The live contract fetched on 2026-09-04 had the same SHA-256.

The CLI validates all seven MVP public API operations:

1. `GET /forms`
2. `POST /forms`
3. `GET /forms/{formId}`
4. `PATCH /forms/{formId}`
5. `GET /forms/{formId}/fields`
6. `PUT /forms/{formId}/fields`
7. `GET /forms/{formId}/snippet`

## Contract generation

Generated models are committed at:

```text
internal/api/models.gen.go
```

Generation uses:

```text
github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen v2.8.0
```

Configuration is committed at:

```text
api/oapi-codegen.yaml
```

Generated nullable properties use explicit nullable handling.

The generated model regeneration check produced no diff during this
validation.

## Contract drift detection

CI performs three independent contract checks.

First, it validates the committed snapshot SHA-256.

Second, it regenerates the public API models and fails if the committed
generated model file is stale.

Third, it downloads the current published OpenAPI contract over HTTPS and runs:

```text
oasdiff v1.30.0
```

The compatibility gate uses:

```text
oasdiff breaking --fail-on ERR --allow-external-refs=false
```

Compatible published additions are visible in CI but do not fail the build.

Incompatible changes fail CI.

External OpenAPI references are disabled.

The local compatibility check on 2026-09-04 reported:

```text
No changes detected
```

## Response validation

Public API responses are treated as untrusted network input.

Validation now covers:

- malformed JSON
- missing required top-level response data
- missing required nested response properties
- incompatible response container shapes
- unexpected response content types
- JSON content types with valid parameters
- oversized responses
- interrupted responses
- unexpected server responses

Successful API responses must use the JSON media type.

For error responses, non-JSON bodies are not parsed or exposed.

Required response-property presence is validated before generated Go models
are used.

Additional response properties remain allowed so compatible API additions do
not fail solely because they add fields.

## Credential handling

The API credential is read from:

```text
POSTMYFORM_API_TOKEN
```

The CLI does not provide a normal command-line token argument.

Automated tests use synthetic sentinel credentials.

Tests verify that credentials are not exposed through:

- API error strings
- CLI standard output
- CLI standard error
- transport-failure output
- rate-limit output

The complete bearer value is treated as sensitive because it contains the
credential.

Untrusted server error text is sanitized before it is exposed.

The `Retry-After` response value also passes through credential redaction before
the CLI can print it.

No real production credential is used by these automated tests.

## Authentication and authorization

The CLI preserves server-side authorization behavior.

Automated validation covers public API meanings for:

- missing local credential
- invalid credential / HTTP 401
- insufficient scope / HTTP 403
- inaccessible resource / HTTP 404

The public contract intentionally returns the same not-found response for a
form in another organization, a deleted form, an unknown form, or an invalid
form ID.

The CLI does not attempt to reproduce tenant authorization locally.

The public contract does not define distinct expired-credential or
revoked-credential error codes. Credentials that are not valid use the public
unauthorized behavior.

## Mutation safety

The CLI does not automatically retry mutations.

Explicit tests verify one request attempt for:

- create form
- update form
- replace form fields

A failed or ambiguous mutation response does not cause an automatic second
mutation request.

## Rate limiting

HTTP 429 responses remain API failures.

If the public API supplies `Retry-After`, the CLI preserves and prints that
value after credential redaction.

The CLI does not invent a retry value.

Mutations are not automatically retried.

## Network behavior

The client uses bounded HTTP timeouts and bounded response sizes.

Validation covers:

- connection failure
- timeout
- TLS certificate verification failure
- server 5xx response
- interrupted response
- oversized response

Remote API endpoints require HTTPS.

Plain HTTP is allowed only for loopback test endpoints.

Normal TLS certificate verification is not disabled.

The operating system trust store is used.

## CLI security behavior

Existing and expanded tests validate that:

- JSON mode writes machine-readable JSON to stdout
- errors remain on stderr
- failures return non-zero exit status
- credentials are not accepted through a normal token argument
- structured field input is parsed as data
- unsupported structured properties are rejected
- structured input is size bounded
- API values are not executed as shell commands
- generated snippets are returned as data and are not executed
- mutations are not automatically retried

## Dependency security

CI uses:

```text
govulncheck v1.7.0
```

Direct and transitive Go dependencies used by the built program are in scope.

Any vulnerability reported by the pinned `govulncheck` gate is
release-blocking.

There is no CVSS threshold that permits a reported reachable vulnerability to
pass.

Remediation should update, replace, or remove the affected dependency.

An exception requires:

- documented technical review
- a tracked issue
- an explicit release decision

Unresolved release-blocking dependency findings fail CI and must not be
released.

Local result on 2026-09-04:

```text
No vulnerabilities found.
```

Release builds also generate CycloneDX JSON SBOMs.

## Platform validation

The supported native targets are:

```text
linux/amd64
linux/arm64
darwin/amd64
darwin/arm64
windows/amd64
```

The existing CI cross-build matrix validates all five targets.

The `v0.1.1` release produced binaries and CycloneDX SBOMs for all five targets.

Release checksum verification passed for every binary and SBOM.

## Local validation results

The following local gates passed on 2026-09-04:

```text
gofmt
go vet ./...
go test ./...
go test -race ./...
pinned OpenAPI SHA-256 verification
oapi-codegen v2.8.0 regenerate-and-diff
oasdiff v1.30.0 live compatibility check
govulncheck v1.7.0
git diff --check
```

The live OpenAPI compatibility check reported no changes.

The vulnerability check reported no vulnerabilities.

## CLI staging smoke

A controlled CLI staging smoke was completed on 2026-09-04 through the public
PostMyForm API boundary.

Synthetic data only was used.

The smoke covered:

1. authenticate
2. list forms
3. create a synthetic form
4. retrieve the synthetic form
5. update one supported property
6. replace form fields
7. retrieve form fields
8. retrieve the generated snippet

The synthetic form was clearly identified as CLI staging smoke data.

The retained synthetic form was paused after validation because the supported
public API does not provide a delete operation.

No private route or database operation was used for cleanup.

The smoke used normal public mutation paths. The CLI does not provide a path
that bypasses normal server-side mutation auditing.

## Staging credential status

The completed CLI staging smoke used a staging API credential.

This validation has not yet established that the credential is the dedicated
least-privilege developer-tooling credential required for the final
cross-product validation.

Dedicated least-privilege credential designation remains a required closeout
item before the complete developer-tooling validation story can pass.

No credential value is recorded in this document.

## Pending cross-product validation

The following items remain intentionally pending until the PostMyForm MCP
implementation is available:

- MCP credential sentinel tests
- MCP argument and tool security validation
- proof that no generic MCP HTTP proxy exists
- MCP mutation identification validation
- CLI/MCP error-category compatibility
- dedicated least-privilege cross-product staging credential validation
- MCP protocol staging smoke
- final cross-product staging evidence

These pending items do not reduce the completed CLI validation recorded above.

## Known limitations

The CLI and MCP are independent products and do not share a runtime client
library.

The live contract compatibility check requires network access to the public
PostMyForm OpenAPI endpoint.

A temporary network failure while fetching the authoritative contract fails the
CI compatibility step rather than silently skipping validation.

The public API does not expose a form-delete operation, so synthetic staging
forms can require documented retention rather than API cleanup.
