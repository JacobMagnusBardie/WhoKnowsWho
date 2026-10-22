package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestAPIRegisterPasswordConfirmation(t *testing.T) {
	for _, contentType := range []string{"application/x-www-form-urlencoded", "application/json"} {
		for _, confirmation := range []struct {
			name       string
			password2  string
			wantStatus int
		}{
			{name: "mismatched", password2: "different-password", wantStatus: http.StatusUnprocessableEntity},
			{name: "matching", password2: "password123", wantStatus: http.StatusOK},
			{name: "omitted", wantStatus: http.StatusOK},
		} {
			t.Run(contentType+"/"+confirmation.name, func(t *testing.T) {
				conn, err := sql.Open("sqlite", ":memory:")
				if err != nil {
					t.Fatalf("open test database: %v", err)
				}
				conn.SetMaxOpenConns(1)
				previousDB := db
				db = conn
				t.Cleanup(func() {
					db = previousDB
					if err := conn.Close(); err != nil {
						t.Errorf("close test database: %v", err)
					}
				})
				if _, err := conn.ExecContext(t.Context(), schema); err != nil {
					t.Fatalf("create test schema: %v", err)
				}

				fields := map[string]string{
					"username": "newuser",
					"email":    "newuser@example.com",
					"password": "password123",
				}
				if confirmation.password2 != "" {
					fields["password2"] = confirmation.password2
				}
				var body string
				if contentType == "application/json" {
					encoded, err := json.Marshal(fields)
					if err != nil {
						t.Fatalf("encode request: %v", err)
					}
					body = string(encoded)
				} else {
					form := url.Values{}
					for name, value := range fields {
						form.Set(name, value)
					}
					body = form.Encode()
				}
				request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/register", strings.NewReader(body))
				request.Header.Set("Content-Type", contentType)
				response := httptest.NewRecorder()

				apiRegister(response, request)

				if response.Code != confirmation.wantStatus {
					t.Fatalf("status = %d, want %d: %s", response.Code, confirmation.wantStatus, response.Body.String())
				}
				if got := response.Header().Get("Content-Type"); got != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", got)
				}
				if confirmation.wantStatus == http.StatusUnprocessableEntity {
					var validation HTTPValidationError
					if err := json.Unmarshal(response.Body.Bytes(), &validation); err != nil {
						t.Fatalf("decode validation response: %v", err)
					}
					wantDetail := []ValidationError{
						{Loc: []interface{}{"body"}, Msg: "The two passwords do not match", Type: "value_error"},
					}
					if !reflect.DeepEqual(validation.Detail, wantDetail) {
						t.Errorf("detail = %#v, want %#v", validation.Detail, wantDetail)
					}
					var count int
					if err := conn.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
						t.Fatalf("count users: %v", err)
					}
					if count != 0 {
						t.Errorf("user count = %d, want 0 after rejected registration", count)
					}
					return
				}

				var passwordHash string
				if err := conn.QueryRowContext(t.Context(), "SELECT password FROM users WHERE username = ?", fields["username"]).Scan(&passwordHash); err != nil {
					t.Fatalf("query registered user: %v", err)
				}
				if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(fields["password"])); err != nil {
					t.Errorf("stored password does not match: %v", err)
				}
			})
		}
	}
}

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
