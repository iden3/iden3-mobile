package iden3mobile

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/dghubble/sling"
	"github.com/iden3/go-iden3-core/components/httpclient"
	"gopkg.in/go-playground/validator.v9"
)

const (
	fieldErrMsg = "Key: '%s' Error:Field validation for '%s' failed on the '%s' tag"
)

func NewValidationError(validationErrors validator.ValidationErrors) ValidationError {
	n := len(validationErrors)
	err := ValidationError{FieldError: validationErrors[n-1], Next: nil}
	for i := n - 2; i > 0; i-- {
		err = ValidationError{FieldError: validationErrors[i], Next: &err}
	}
	return err
}

type ValidationError struct {
	FieldError validator.FieldError
	Next       *ValidationError
}

func (e ValidationError) Error() string {
	buff := bytes.NewBufferString("")
	ve := &e
	for {
		fe := ve.FieldError
		buff.WriteString(fmt.Sprintf(fieldErrMsg, fe.Namespace(), fe.Field(), fe.Tag()))
		buff.WriteString("\n")
		ve = ve.Next
		if ve == nil {
			break
		}
	}
	return strings.TrimSpace(buff.String())
}

type HttpClient struct {
	httpclient.HttpClient
}

func NewHttpClient(urlBase string) *HttpClient {
	return &HttpClient{HttpClient: *httpclient.NewHttpClient(urlBase)}
}

func (p *HttpClient) DoRequest(s *sling.Sling, res interface{}) error {
	err := p.HttpClient.DoRequest(s, res)
	switch e := err.(type) {
	case validator.ValidationErrors:
		return NewValidationError(e)
	default:
		return err
	}
}
