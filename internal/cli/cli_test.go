package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PostMyForm/postmyform-cli/internal/api"
)

func TestHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"help"}, &stdout, &stderr)

	if code != ExitSuccess {
		t.Fatalf("exit code = %d, want %d", code, ExitSuccess)
	}
	if !strings.Contains(stdout.String(), "postmyform forms <command>") {
		t.Fatalf("stdout missing usage: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"version"}, &stdout, &stderr)

	if code != ExitSuccess {
		t.Fatalf("exit code = %d, want %d", code, ExitSuccess)
	}
	if stdout.String() != "dev\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "dev\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestUnknownCommandUsesStderrAndUsageExit(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"unknown"}, &stdout, &stderr)

	if code != ExitUsage {
		t.Fatalf("exit code = %d, want %d", code, ExitUsage)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr missing error: %q", stderr.String())
	}
}

func TestFormsListRequiresCredential(t *testing.T) {
	t.Setenv("POSTMYFORM_API_TOKEN", "")
	t.Setenv("POSTMYFORM_API_BASE_URL", "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"forms", "list"}, &stdout, &stderr)

	if code != ExitAuth {
		t.Fatalf("exit code = %d, want %d", code, ExitAuth)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "POSTMYFORM_API_TOKEN is required") {
		t.Fatalf("stderr = %q, want missing credential error", stderr.String())
	}
}

func TestFormsListHumanOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data":[{
				"id":"11111111-1111-1111-1111-111111111111",
				"name":"Contact",
				"slug":"contact",
				"endpointId":"endpoint-1",
				"status":"active",
				"destinationEmail":"owner@example.com",
				"allowedOrigins":[],
				"successRedirectUrl":null,
				"spamHoneypotField":"spamField",
				"minSubmitSeconds":3,
				"submissionUrl":"https://example.invalid/f/endpoint-1",
				"createdAt":"2026-09-03T12:00:00Z",
				"updatedAt":"2026-09-03T12:00:00Z"
			}]
		}`))
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", "test-token")
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"forms", "list"}, &stdout, &stderr)

	if code != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr = %q", code, ExitSuccess, stderr.String())
	}
	if got := stdout.String(); got != "11111111-1111-1111-1111-111111111111\tactive\tContact\n" {
		t.Fatalf("stdout = %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestFormsListJSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", "test-token")
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"forms", "list", "--json"}, &stdout, &stderr)

	if code != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr = %q", code, ExitSuccess, stderr.String())
	}

	var output struct {
		Forms []api.Form `json:"forms"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid JSON: %v; output = %q", err, stdout.String())
	}
	if output.Forms == nil {
		t.Fatal("forms JSON field is nil, want stable empty array")
	}
	if len(output.Forms) != 0 {
		t.Fatalf("forms length = %d, want 0", len(output.Forms))
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestFormsGetRejectsInvalidFormID(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"forms", "get", "not-a-uuid"}, &stdout, &stderr)

	if code != ExitUsage {
		t.Fatalf("exit code = %d, want %d", code, ExitUsage)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "form ID must be a valid UUID") {
		t.Fatalf("stderr = %q, want UUID validation error", stderr.String())
	}
}

func TestFormsGetHumanOutput(t *testing.T) {
	const formID = "11111111-1111-1111-1111-111111111111"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/forms/"+formID {
			t.Fatalf("path = %q, want %q", got, "/forms/"+formID)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data":{
				"id":"11111111-1111-1111-1111-111111111111",
				"name":"Contact",
				"slug":"contact",
				"endpointId":"endpoint-1",
				"status":"active",
				"destinationEmail":"owner@example.com",
				"allowedOrigins":[],
				"successRedirectUrl":null,
				"spamHoneypotField":"spamField",
				"minSubmitSeconds":3,
				"submissionUrl":"https://example.invalid/f/endpoint-1",
				"createdAt":"2026-09-03T12:00:00Z",
				"updatedAt":"2026-09-03T12:00:00Z"
			}
		}`))
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", "test-token")
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"forms", "get", formID}, &stdout, &stderr)

	if code != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr = %q", code, ExitSuccess, stderr.String())
	}

	want := "" +
		"ID: 11111111-1111-1111-1111-111111111111\n" +
		"Name: Contact\n" +
		"Status: active\n" +
		"Destination: owner@example.com\n" +
		"Submission URL: https://example.invalid/f/endpoint-1\n"

	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestFormsCreateHumanOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/forms" {
			t.Fatalf("path = %q, want /forms", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if body["name"] != "Contact" {
			t.Fatalf("name = %#v, want Contact", body["name"])
		}
		if body["destinationEmail"] != "owner@example.com" {
			t.Fatalf("destinationEmail = %#v", body["destinationEmail"])
		}
		if body["successRedirectUrl"] != "https://example.com/thanks" {
			t.Fatalf("successRedirectUrl = %#v", body["successRedirectUrl"])
		}

		origins, ok := body["allowedOrigins"].([]any)
		if !ok || len(origins) != 2 {
			t.Fatalf("allowedOrigins = %#v, want 2 values", body["allowedOrigins"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"data":{
				"id":"11111111-1111-1111-1111-111111111111",
				"name":"Contact",
				"slug":"contact",
				"endpointId":"endpoint-1",
				"status":"active",
				"destinationEmail":"owner@example.com",
				"allowedOrigins":["https://example.com","https://www.example.com"],
				"successRedirectUrl":"https://example.com/thanks",
				"spamHoneypotField":"spamField",
				"minSubmitSeconds":3,
				"submissionUrl":"https://postmyform.com/f/endpoint-1",
				"createdAt":"2026-09-03T12:00:00Z",
				"updatedAt":"2026-09-03T12:00:00Z"
			}
		}`))
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", "test-token")
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"forms", "create",
		"--name", "Contact",
		"--destination-email", "owner@example.com",
		"--allowed-origin", "https://example.com",
		"--allowed-origin", "https://www.example.com",
		"--success-redirect-url", "https://example.com/thanks",
	}, &stdout, &stderr)

	if code != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr = %q", code, ExitSuccess, stderr.String())
	}

	want := "" +
		"Created form: 11111111-1111-1111-1111-111111111111\n" +
		"Name: Contact\n" +
		"Status: active\n" +
		"Submission URL: https://postmyform.com/f/endpoint-1\n"

	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestFormsCreateRequiresName(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"forms", "create",
		"--destination-email", "owner@example.com",
	}, &stdout, &stderr)

	if code != ExitUsage {
		t.Fatalf("exit code = %d, want %d", code, ExitUsage)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "--name is required") {
		t.Fatalf("stderr = %q, want missing name error", stderr.String())
	}
}

func TestFormsCreateJSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"data":{
				"id":"11111111-1111-1111-1111-111111111111",
				"name":"Contact",
				"slug":"contact",
				"endpointId":"endpoint-1",
				"status":"active",
				"destinationEmail":"owner@example.com",
				"allowedOrigins":[],
				"successRedirectUrl":null,
				"spamHoneypotField":"spamField",
				"minSubmitSeconds":3,
				"submissionUrl":"https://postmyform.com/f/endpoint-1",
				"createdAt":"2026-09-03T12:00:00Z",
				"updatedAt":"2026-09-03T12:00:00Z"
			}
		}`))
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", "test-token")
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"forms", "create",
		"--name", "Contact",
		"--destination-email", "owner@example.com",
		"--json",
	}, &stdout, &stderr)

	if code != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr = %q", code, ExitSuccess, stderr.String())
	}

	var output api.Form
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid JSON: %v; output = %q", err, stdout.String())
	}
	if output.Id.String() != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("form ID = %q", output.Id.String())
	}
	if output.Name != "Contact" {
		t.Fatalf("form name = %q, want Contact", output.Name)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestFormsUpdateSendsOnlyChangedProperty(t *testing.T) {
	const formID = "11111111-1111-1111-1111-111111111111"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("method = %q, want PATCH", r.Method)
		}
		if r.URL.Path != "/forms/"+formID {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/forms/"+formID)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if len(body) != 1 {
			t.Fatalf("request body = %#v, want exactly one property", body)
		}
		if body["name"] != "Updated Contact" {
			t.Fatalf("name = %#v, want Updated Contact", body["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data":{
				"id":"11111111-1111-1111-1111-111111111111",
				"status":"active",
				"updatedAt":"2026-09-03T12:00:00Z"
			}
		}`))
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", "test-token")
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"forms", "update", formID,
		"--name", "Updated Contact",
	}, &stdout, &stderr)

	if code != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr = %q", code, ExitSuccess, stderr.String())
	}

	want := "" +
		"Updated form: 11111111-1111-1111-1111-111111111111\n" +
		"Status: active\n" +
		"Updated at: 2026-09-03T12:00:00Z\n"

	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestFormsUpdateExplicitClear(t *testing.T) {
	const formID = "11111111-1111-1111-1111-111111111111"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		origins, ok := body["allowedOrigins"].([]any)
		if !ok {
			t.Fatalf("allowedOrigins = %#v, want array", body["allowedOrigins"])
		}
		if len(origins) != 0 {
			t.Fatalf("allowedOrigins = %#v, want empty array", origins)
		}

		redirect, exists := body["successRedirectUrl"]
		if !exists {
			t.Fatal("successRedirectUrl is omitted, want explicit null")
		}
		if redirect != nil {
			t.Fatalf("successRedirectUrl = %#v, want null", redirect)
		}

		if len(body) != 2 {
			t.Fatalf("request body = %#v, want exactly two properties", body)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data":{
				"id":"11111111-1111-1111-1111-111111111111",
				"status":"active",
				"updatedAt":"2026-09-03T12:00:00Z"
			}
		}`))
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", "test-token")
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"forms", "update", formID,
		"--clear-allowed-origins",
		"--clear-success-redirect-url",
		"--json",
	}, &stdout, &stderr)

	if code != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr = %q", code, ExitSuccess, stderr.String())
	}

	var output api.FormMutationReceipt
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid JSON: %v; output = %q", err, stdout.String())
	}
	if output.Id.String() != formID {
		t.Fatalf("form ID = %q, want %q", output.Id.String(), formID)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestFormsFieldsGetHumanOutput(t *testing.T) {
	const formID = "11111111-1111-1111-1111-111111111111"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/forms/"+formID+"/fields" {
			t.Fatalf("path = %q", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data":[
				{
					"name":"email",
					"label":"Email",
					"fieldType":"email",
					"required":true,
					"options":null
				},
				{
					"name":"message",
					"label":"Message",
					"fieldType":"textarea",
					"required":false,
					"options":null
				}
			]
		}`))
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", "test-token")
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"forms", "fields", "get", formID,
	}, &stdout, &stderr)

	if code != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr = %q", code, ExitSuccess, stderr.String())
	}

	want := "" +
		"email\temail\tEmail\ttrue\n" +
		"message\ttextarea\tMessage\tfalse\n"

	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestFormsFieldsGetJSONOutput(t *testing.T) {
	const formID = "11111111-1111-1111-1111-111111111111"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data":[
				{
					"name":"email",
					"label":"Email",
					"fieldType":"email",
					"required":true,
					"options":null
				}
			]
		}`))
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", "test-token")
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"forms", "fields", "get", formID, "--json",
	}, &stdout, &stderr)

	if code != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr = %q", code, ExitSuccess, stderr.String())
	}

	var output struct {
		Fields []api.FormField `json:"fields"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid JSON: %v; output = %q", err, stdout.String())
	}
	if len(output.Fields) != 1 {
		t.Fatalf("fields length = %d, want 1", len(output.Fields))
	}
	if output.Fields[0].Name != "email" {
		t.Fatalf("field name = %q, want email", output.Fields[0].Name)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestFormsFieldsReplaceFromStdin(t *testing.T) {
	const formID = "11111111-1111-1111-1111-111111111111"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %q, want PUT", r.Method)
		}
		if r.URL.Path != "/forms/"+formID+"/fields" {
			t.Fatalf("path = %q", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		fields, ok := body["fields"].([]any)
		if !ok {
			t.Fatalf("fields = %#v, want array", body["fields"])
		}
		if len(fields) != 2 {
			t.Fatalf("fields length = %d, want 2", len(fields))
		}

		first, ok := fields[0].(map[string]any)
		if !ok {
			t.Fatalf("first field = %#v", fields[0])
		}
		if first["name"] != "email" {
			t.Fatalf("first field name = %#v, want email", first["name"])
		}
		if first["fieldType"] != "email" {
			t.Fatalf("first field type = %#v, want email", first["fieldType"])
		}
		if first["options"] != nil {
			t.Fatalf("first field options = %#v, want null", first["options"])
		}

		second, ok := fields[1].(map[string]any)
		if !ok {
			t.Fatalf("second field = %#v", fields[1])
		}
		options, ok := second["options"].([]any)
		if !ok || len(options) != 2 {
			t.Fatalf("second field options = %#v, want 2 values", second["options"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data":[
				{
					"name":"email",
					"label":"Email",
					"fieldType":"email",
					"required":true,
					"options":null
				},
				{
					"name":"topic",
					"label":"Topic",
					"fieldType":"select",
					"required":false,
					"options":["Sales","Support"]
				}
			]
		}`))
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", "test-token")
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	stdin := strings.NewReader(`{
		"fields":[
			{
				"name":"email",
				"label":"Email",
				"fieldType":"email",
				"required":true,
				"options":null
			},
			{
				"name":"topic",
				"label":"Topic",
				"fieldType":"select",
				"required":false,
				"options":["Sales","Support"]
			}
		]
	}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runFormsFieldsReplaceWithStdin(
		[]string{formID, "--file", "-"},
		stdin,
		&stdout,
		&stderr,
	)

	if code != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr = %q", code, ExitSuccess, stderr.String())
	}

	want := "Replaced 2 fields for form: " + formID + "\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestFormsSnippetHumanOutput(t *testing.T) {
	const formID = "11111111-1111-1111-1111-111111111111"
	const html = `<form action="https://postmyform.com/f/endpoint-1" method="post"><input name="email"></form>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/forms/"+formID+"/snippet" {
			t.Fatalf("path = %q", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"data":{"html":%q}}`, html)
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", "test-token")
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(
		[]string{"forms", "snippet", formID},
		&stdout,
		&stderr,
	)

	if code != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr = %q", code, ExitSuccess, stderr.String())
	}
	if stdout.String() != html {
		t.Fatalf("stdout = %q, want %q", stdout.String(), html)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestFormsSnippetJSONOutput(t *testing.T) {
	const formID = "11111111-1111-1111-1111-111111111111"
	const html = `<form action="/f/example"></form>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"data":{"html":%q}}`, html)
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", "test-token")
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(
		[]string{"forms", "snippet", formID, "--json"},
		&stdout,
		&stderr,
	)

	if code != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr = %q", code, ExitSuccess, stderr.String())
	}

	var output struct {
		HTML string `json:"html"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid JSON: %v; output = %q", err, stdout.String())
	}
	if output.HTML != html {
		t.Fatalf("html = %q, want %q", output.HTML, html)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestFormsListNetworkFailureUsesNetworkExitAndRedactsToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	baseURL := server.URL
	server.Close()

	const token = "super-secret-test-token"

	t.Setenv("POSTMYFORM_API_TOKEN", token)
	t.Setenv("POSTMYFORM_API_BASE_URL", baseURL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"forms", "list"}, &stdout, &stderr)

	if code != ExitNetwork {
		t.Fatalf("exit code = %d, want %d", code, ExitNetwork)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if got := stderr.String(); got != "unable to reach the PostMyForm API\n" {
		t.Fatalf("stderr = %q", got)
	}
	if strings.Contains(stderr.String(), token) {
		t.Fatal("stderr contains API token")
	}
}

func TestFormsListRateLimitShowsRetryAfter(t *testing.T) {
	const token = "rate-limit-secret-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{
			"error":{
				"code":"rate_limited",
				"message":"Too many requests"
			}
		}`))
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", token)
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"forms", "list"}, &stdout, &stderr)

	if code != ExitAPI {
		t.Fatalf("exit code = %d, want %d", code, ExitAPI)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	want := "" +
		"Too many requests\n" +
		"Retry-After: 30\n"

	if stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
	if strings.Contains(stderr.String(), token) {
		t.Fatal("stderr contains API token")
	}
}

func TestFormsListAuthFailuresUseAuthExit(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "unauthorized", statusCode: http.StatusUnauthorized},
		{name: "forbidden", statusCode: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(`{
					"error":{
						"code":"unauthorized",
						"message":"Access denied"
					}
				}`))
			}))
			defer server.Close()

			t.Setenv("POSTMYFORM_API_TOKEN", "test-token")
			t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run([]string{"forms", "list"}, &stdout, &stderr)

			if code != ExitAuth {
				t.Fatalf("exit code = %d, want %d", code, ExitAuth)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if stderr.String() != "Access denied\n" {
				t.Fatalf("stderr = %q, want Access denied", stderr.String())
			}
		})
	}
}

func TestFormsListMalformedSuccessFailsClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":`))
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", "test-token")
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"forms", "list"}, &stdout, &stderr)

	if code != ExitAPI {
		t.Fatalf("exit code = %d, want %d", code, ExitAPI)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "malformed response") {
		t.Fatalf("stderr = %q, want malformed response error", stderr.String())
	}
}

func TestFormsGetJSONOutput(t *testing.T) {
	const formID = "11111111-1111-1111-1111-111111111111"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/forms/"+formID {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/forms/"+formID)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data":{
				"id":"11111111-1111-1111-1111-111111111111",
				"name":"Contact",
				"slug":"contact",
				"endpointId":"endpoint-1",
				"status":"active",
				"destinationEmail":"owner@example.com",
				"allowedOrigins":[],
				"successRedirectUrl":null,
				"spamHoneypotField":"spamField",
				"minSubmitSeconds":3,
				"submissionUrl":"https://postmyform.com/f/endpoint-1",
				"createdAt":"2026-09-03T12:00:00Z",
				"updatedAt":"2026-09-03T12:00:00Z"
			}
		}`))
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", "test-token")
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(
		[]string{"forms", "get", formID, "--json"},
		&stdout,
		&stderr,
	)

	if code != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr = %q", code, ExitSuccess, stderr.String())
	}

	var output api.Form
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid JSON: %v; output = %q", err, stdout.String())
	}
	if output.Id.String() != formID {
		t.Fatalf("form ID = %q, want %q", output.Id.String(), formID)
	}
	if output.Name != "Contact" {
		t.Fatalf("form name = %q, want Contact", output.Name)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestFormsCreateLocalValidationUsesUsageExit(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"forms", "create",
		"--name", "Contact",
		"--destination-email", "invalid",
	}, &stdout, &stderr)

	if code != ExitUsage {
		t.Fatalf("exit code = %d, want %d", code, ExitUsage)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.String() != "invalid local request\n" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestFormsCreateServerValidationUsesAPIExit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{
			"error":{
				"code":"invalid_request",
				"message":"form name is not allowed"
			}
		}`))
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", "test-token")
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"forms", "create",
		"--name", "Contact",
		"--destination-email", "owner@example.com",
	}, &stdout, &stderr)

	if code != ExitAPI {
		t.Fatalf("exit code = %d, want %d", code, ExitAPI)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.String() != "form name is not allowed\n" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestFormsListServerFailureRedactsToken(t *testing.T) {
	const token = "server-error-secret-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, `{
			"error":{
				"code":"organization_unavailable",
				"message":"internal failure involving %s"
			}
		}`, token)
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", token)
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"forms", "list"}, &stdout, &stderr)

	if code != ExitAPI {
		t.Fatalf("exit code = %d, want %d", code, ExitAPI)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	want := "internal failure involving [REDACTED]\n"
	if stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
	if strings.Contains(stderr.String(), token) {
		t.Fatal("stderr contains API token")
	}
}

func TestFormsUpdateLocalValidationUsesUsageExit(t *testing.T) {
	const formID = "11111111-1111-1111-1111-111111111111"

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"forms", "update", formID,
		"--destination-email", "invalid",
	}, &stdout, &stderr)

	if code != ExitUsage {
		t.Fatalf("exit code = %d, want %d", code, ExitUsage)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.String() != "invalid local request\n" {
		t.Fatalf("stderr = %q, want invalid local request", stderr.String())
	}
}

func TestFormsFieldsReplaceJSONOutput(t *testing.T) {
	const formID = "11111111-1111-1111-1111-111111111111"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %q, want PUT", r.Method)
		}
		if r.URL.Path != "/forms/"+formID+"/fields" {
			t.Fatalf("path = %q", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data":[
				{
					"name":"email",
					"label":"Email",
					"fieldType":"email",
					"required":true,
					"options":null
				}
			]
		}`))
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", "test-token")
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	stdin := strings.NewReader(`{
		"fields":[
			{
				"name":"email",
				"label":"Email",
				"fieldType":"email",
				"required":true,
				"options":null
			}
		]
	}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runFormsFieldsReplaceWithStdin(
		[]string{formID, "--file", "-", "--json"},
		stdin,
		&stdout,
		&stderr,
	)

	if code != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr = %q", code, ExitSuccess, stderr.String())
	}

	var output struct {
		FormID string          `json:"formId"`
		Fields []api.FormField `json:"fields"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid JSON: %v; output = %q", err, stdout.String())
	}
	if output.FormID != formID {
		t.Fatalf("formId = %q, want %q", output.FormID, formID)
	}
	if len(output.Fields) != 1 {
		t.Fatalf("fields length = %d, want 1", len(output.Fields))
	}
	if output.Fields[0].Name != "email" {
		t.Fatalf("field name = %q, want email", output.Fields[0].Name)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestFormsFieldsReplaceRejectsUnknownJSONProperty(t *testing.T) {
	const formID = "11111111-1111-1111-1111-111111111111"

	stdin := strings.NewReader(`{
		"fields":[],
		"unexpected":true
	}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runFormsFieldsReplaceWithStdin(
		[]string{formID, "--file", "-"},
		stdin,
		&stdout,
		&stderr,
	)

	if code != ExitUsage {
		t.Fatalf("exit code = %d, want %d", code, ExitUsage)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.String() != "structured input must be a valid fields replacement request\n" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestFormsFieldsReplaceRejectsInvalidFieldType(t *testing.T) {
	const formID = "11111111-1111-1111-1111-111111111111"

	stdin := strings.NewReader(`{
		"fields":[
			{
				"name":"example",
				"label":"Example",
				"fieldType":"invalid",
				"required":false,
				"options":null
			}
		]
	}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runFormsFieldsReplaceWithStdin(
		[]string{formID, "--file", "-"},
		stdin,
		&stdout,
		&stderr,
	)

	if code != ExitUsage {
		t.Fatalf("exit code = %d, want %d", code, ExitUsage)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.String() != "field 0 has invalid fieldType \"invalid\"\n" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestFormsListAPIFailuresUseAPIExit(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		code       string
		message    string
	}{
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			code:       "not_found",
			message:    "Resource not found",
		},
		{
			name:       "conflict",
			statusCode: http.StatusConflict,
			code:       "conflict",
			message:    "Request conflicts with current state",
		},
		{
			name:       "server failure",
			statusCode: http.StatusInternalServerError,
			code:       "internal_error",
			message:    "Internal server error",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = fmt.Fprintf(
					w,
					`{"error":{"code":%q,"message":%q}}`,
					tt.code,
					tt.message,
				)
			}))
			defer server.Close()

			t.Setenv("POSTMYFORM_API_TOKEN", "test-token")
			t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run([]string{"forms", "list"}, &stdout, &stderr)

			if code != ExitAPI {
				t.Fatalf("exit code = %d, want %d", code, ExitAPI)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if stderr.String() != tt.message+"\n" {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.message+"\n")
			}
		})
	}
}

func TestFormsListRateLimitRedactsCredentialFromRetryAfter(t *testing.T) {
	const token = "777777"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", token)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{
			"error":{
				"code":"rate_limited",
				"message":"Too many requests"
			}
		}`))
	}))
	defer server.Close()

	t.Setenv("POSTMYFORM_API_TOKEN", token)
	t.Setenv("POSTMYFORM_API_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"forms", "list"}, &stdout, &stderr)

	if code != ExitAPI {
		t.Fatalf("exit code = %d, want %d", code, ExitAPI)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if strings.Contains(stderr.String(), token) {
		t.Fatalf("stderr exposed API credential: %q", stderr.String())
	}

	want := "" +
		"Too many requests\n" +
		"Retry-After: [REDACTED]\n"

	if stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
}
