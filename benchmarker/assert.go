package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
)

// 検証用コード群

func assertInitialize(step *isucandar.BenchmarkStep, res *http.Response) {
	err := assertStatusCode(res, 200)
	if err != nil {
		step.AddError(failure.NewError(ErrCritical, err))
	}
}

func assertStatusCode(res *http.Response, code int) error {
	if res.StatusCode != code {
		return failure.NewError(ErrInvalidStatusCode, fmt.Errorf("Invalid status code: %d (expected: %d)", res.StatusCode, code))
	}
	return nil
}

func assertContentType(res *http.Response, contentType string) error {
	actual := res.Header.Get("Content-Type")
	if !strings.HasPrefix(actual, contentType) {
		return failure.NewError(ErrInvalidContentType, fmt.Errorf("Invalid content type: %s (expected: %s)", actual, contentType))
	}
	return nil
}

func assertJSONBody(res *http.Response, body interface{}) error {
	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	if err := decoder.Decode(body); err != nil {
		return failure.NewError(ErrInvalidJSON, fmt.Errorf("Invalid JSON"))
	}
	return nil
}
