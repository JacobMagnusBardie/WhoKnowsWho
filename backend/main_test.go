package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestAPISearchRejectsMissingQuery(t *testing.T) {
	for _, target := range []string{"/api/search", "/api/search?q="} {
		t.Run(target, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)

			apiSearch(response, request)

			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422", response.Code)
			}
			if got := response.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}

			var body struct {
				StatusCode int               `json:"statusCode"`
				Message    string            `json:"message"`
				Detail     []ValidationError `json:"detail"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.StatusCode != http.StatusUnprocessableEntity {
				t.Errorf("body statusCode = %d, want 422", body.StatusCode)
			}
			if !strings.Contains(body.Message, "q") {
				t.Errorf("message = %q, want an explanation naming the q parameter", body.Message)
			}

			wantDetail := []ValidationError{
				{Loc: []interface{}{"query", "q"}, Msg: "Field required", Type: "missing"},
			}
			if !reflect.DeepEqual(body.Detail, wantDetail) {
				t.Errorf("detail = %#v, want existing validation detail %#v", body.Detail, wantDetail)
			}
		})
	}
}
