package api

import (
	"errors"
	"testing"
)

func TestValidateResponseShapeRejectsIncompleteResponses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		body   string
		result any
	}{
		{
			name:   "form response missing required form field",
			body:   `{"data":{"id":"11111111-1111-1111-1111-111111111111"}}`,
			result: &FormResponse{},
		},
		{
			name:   "mutation response missing updatedAt",
			body:   `{"data":{"id":"11111111-1111-1111-1111-111111111111","status":"active"}}`,
			result: &FormMutationResponse{},
		},
		{
			name:   "fields response missing options",
			body:   `{"data":[{"name":"email","label":"Email","fieldType":"email","required":true}]}`,
			result: &FormFieldsResponse{},
		},
		{
			name:   "snippet response missing html",
			body:   `{"data":{}}`,
			result: &FormSnippetResponse{},
		},
		{
			name:   "list response has incompatible data shape",
			body:   `{"data":{}}`,
			result: &FormListResponse{},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := validateResponseShape([]byte(test.body), test.result)
			if !errors.Is(err, ErrMalformedResponse) {
				t.Fatalf("validateResponseShape() error = %v, want %v", err, ErrMalformedResponse)
			}
		})
	}
}

func TestValidateResponseShapeAllowsAdditionalProperties(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"data": {
			"id": "11111111-1111-1111-1111-111111111111",
			"name": "Example",
			"slug": "example",
			"status": "active",
			"endpointId": "endpoint",
			"submissionUrl": "https://example.com/f/endpoint",
			"destinationEmail": "test@example.com",
			"allowedOrigins": [],
			"successRedirectUrl": null,
			"spamHoneypotField": "honeypot",
			"minSubmitSeconds": 1,
			"createdAt": "2026-09-04T00:00:00Z",
			"updatedAt": "2026-09-04T00:00:00Z",
			"futureCompatibleProperty": "allowed"
		},
		"futureEnvelopeProperty": true
	}`)

	if err := validateResponseShape(body, &FormResponse{}); err != nil {
		t.Fatalf("validateResponseShape() error = %v, want nil", err)
	}
}
