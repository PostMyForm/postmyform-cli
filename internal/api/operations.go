package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

type RequestEncodingError struct {
	err error
}

func (e *RequestEncodingError) Error() string {
	return "invalid PostMyForm API request"
}

func (e *RequestEncodingError) Unwrap() error {
	return e.err
}

func (c *Client) ListForms(ctx context.Context) ([]Form, error) {
	var response FormListResponse
	if err := c.do(ctx, http.MethodGet, "/forms", nil, http.StatusOK, &response); err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (c *Client) GetForm(ctx context.Context, formID FormId) (Form, error) {
	var response FormResponse
	if err := c.do(ctx, http.MethodGet, formPath(formID), nil, http.StatusOK, &response); err != nil {
		return Form{}, err
	}

	return response.Data, nil
}

func (c *Client) CreateForm(ctx context.Context, request CreateFormRequest) (Form, error) {
	body, err := encodeJSON(request)
	if err != nil {
		return Form{}, err
	}

	var response FormResponse
	if err := c.do(ctx, http.MethodPost, "/forms", body, http.StatusCreated, &response); err != nil {
		return Form{}, err
	}

	return response.Data, nil
}

func (c *Client) UpdateForm(
	ctx context.Context,
	formID FormId,
	request PatchFormRequest,
) (FormMutationReceipt, error) {
	body, err := encodeJSON(request)
	if err != nil {
		return FormMutationReceipt{}, err
	}

	var response FormMutationResponse
	if err := c.do(
		ctx,
		http.MethodPatch,
		formPath(formID),
		body,
		http.StatusOK,
		&response,
	); err != nil {
		return FormMutationReceipt{}, err
	}

	return response.Data, nil
}

func (c *Client) ListFormFields(ctx context.Context, formID FormId) ([]FormField, error) {
	var response FormFieldsResponse
	if err := c.do(
		ctx,
		http.MethodGet,
		formPath(formID)+"/fields",
		nil,
		http.StatusOK,
		&response,
	); err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (c *Client) ReplaceFormFields(
	ctx context.Context,
	formID FormId,
	request ReplaceFormFieldsRequest,
) ([]FormField, error) {
	body, err := encodeJSON(request)
	if err != nil {
		return nil, err
	}

	var response FormFieldsResponse
	if err := c.do(
		ctx,
		http.MethodPut,
		formPath(formID)+"/fields",
		body,
		http.StatusOK,
		&response,
	); err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (c *Client) GetFormSnippet(ctx context.Context, formID FormId) (FormSnippet, error) {
	var response FormSnippetResponse
	if err := c.do(
		ctx,
		http.MethodGet,
		formPath(formID)+"/snippet",
		nil,
		http.StatusOK,
		&response,
	); err != nil {
		return FormSnippet{}, err
	}

	return response.Data, nil
}

func formPath(formID FormId) string {
	return "/forms/" + url.PathEscape(formID.String())
}

func encodeJSON(value any) (*bytes.Reader, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, &RequestEncodingError{err: err}
	}

	return bytes.NewReader(body), nil
}
