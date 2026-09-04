package api

import (
	"bytes"
	"encoding/json"
	"fmt"
)

var requiredFormResponseFields = []string{
	"id",
	"name",
	"slug",
	"status",
	"endpointId",
	"submissionUrl",
	"destinationEmail",
	"allowedOrigins",
	"successRedirectUrl",
	"spamHoneypotField",
	"minSubmitSeconds",
	"createdAt",
	"updatedAt",
}

var requiredFormFieldResponseFields = []string{
	"name",
	"label",
	"fieldType",
	"required",
	"options",
}

var requiredFormMutationReceiptFields = []string{
	"id",
	"status",
	"updatedAt",
}

var requiredFormSnippetFields = []string{
	"html",
}

func validateResponseShape(body []byte, result any) error {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil || root == nil {
		return ErrMalformedResponse
	}

	data, ok := root["data"]
	if !ok || isJSONNull(data) {
		return fmt.Errorf("%w: missing required field %q", ErrMalformedResponse, "data")
	}

	switch result.(type) {
	case *FormListResponse:
		return validateJSONArrayObjects(data, requiredFormResponseFields)
	case *FormResponse:
		return validateJSONObjectFields(data, requiredFormResponseFields)
	case *FormMutationResponse:
		return validateJSONObjectFields(data, requiredFormMutationReceiptFields)
	case *FormFieldsResponse:
		return validateJSONArrayObjects(data, requiredFormFieldResponseFields)
	case *FormSnippetResponse:
		return validateJSONObjectFields(data, requiredFormSnippetFields)
	default:
		return nil
	}
}

func validateJSONArrayObjects(raw json.RawMessage, required []string) error {
	if isJSONNull(raw) {
		return ErrMalformedResponse
	}

	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return ErrMalformedResponse
	}

	for _, item := range items {
		if err := validateJSONObjectFields(item, required); err != nil {
			return err
		}
	}

	return nil
}

func validateJSONObjectFields(raw json.RawMessage, required []string) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return ErrMalformedResponse
	}

	for _, field := range required {
		if _, ok := object[field]; !ok {
			return fmt.Errorf("%w: missing required field %q", ErrMalformedResponse, field)
		}
	}

	return nil
}

func isJSONNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}
