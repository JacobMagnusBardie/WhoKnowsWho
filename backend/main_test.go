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

// registerCase is one /api/register request in TestAPIRegisterValidation. username and
// password2 override the default fields when set; wantMsg is checked on a 422.
type registerCase struct {
	name       string
	username   string
	password2  string
	wantStatus int
	wantMsg    string
}

func TestAPIRegisterValidation(t *testing.T) {
	cases := []registerCase{
		{name: "mismatched", password2: "different-password", wantStatus: http.StatusUnprocessableEntity, wantMsg: "The two passwords do not match"},
		{name: "matching", password2: "password123", wantStatus: http.StatusOK},
		{name: "omitted", wantStatus: http.StatusOK},
		{name: "username at limit", username: strings.Repeat("a", maxUsernameLength), wantStatus: http.StatusOK},
		{name: "username over limit", username: strings.Repeat("a", maxUsernameLength+1), wantStatus: http.StatusUnprocessableEntity, wantMsg: "The username can be at most 30 characters"},
		{name: "username over limit multibyte", username: strings.Repeat("æ", maxUsernameLength+1), wantStatus: http.StatusUnprocessableEntity, wantMsg: "The username can be at most 30 characters"},
	}
	for _, contentType := range []string{"application/x-www-form-urlencoded", "application/json"} {
		for _, tc := range cases {
			t.Run(contentType+"/"+tc.name, func(t *testing.T) {
				runRegisterCase(t, contentType, tc)
			})
		}
	}
}

// runRegisterCase posts one registration to a fresh database and checks the answer.
func runRegisterCase(t *testing.T, contentType string, tc registerCase) {
	t.Helper()
	conn := newTestDB(t)
	useTestSessionKey(t)
	fields := registerFields(tc)

	response := postRegister(t, contentType, fields)

	if response.Code != tc.wantStatus {
		t.Fatalf("status = %d, want %d: %s", response.Code, tc.wantStatus, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if tc.wantStatus == http.StatusUnprocessableEntity {
		assertRegistrationRejected(t, conn, response, tc.wantMsg)
	} else {
		assertRegisteredAndLoggedIn(t, conn, response, fields)
	}
}

// registerFields returns the form fields for a registration, with the case's overrides applied.
func registerFields(tc registerCase) map[string]string {
	fields := map[string]string{
		"username": "newuser",
		"email":    "newuser@example.com",
		"password": "password123",
	}
	if tc.username != "" {
		fields["username"] = tc.username
	}
	if tc.password2 != "" {
		fields["password2"] = tc.password2
	}
	return fields
}

// postRegister sends fields to apiRegister, encoded as JSON or as a form depending on contentType.
func postRegister(t *testing.T, contentType string, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()
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
	return response
}

// assertRegistrationRejected checks a 422 answer: the validation detail, no user stored
// and no session cookie set.
func assertRegistrationRejected(t *testing.T, conn *sql.DB, response *httptest.ResponseRecorder, wantMsg string) {
	t.Helper()
	var validation HTTPValidationError
	if err := json.Unmarshal(response.Body.Bytes(), &validation); err != nil {
		t.Fatalf("decode validation response: %v", err)
	}
	wantDetail := []ValidationError{
		{Loc: []interface{}{"body"}, Msg: wantMsg, Type: "value_error"},
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
	if cookies := response.Result().Cookies(); len(cookies) != 0 {
		t.Errorf("rejected registration set cookies: %v", cookies)
	}
}

// assertRegisteredAndLoggedIn checks a successful registration: the session cookie logs the
// new user in, and the stored password hash matches the password that was sent.
func assertRegisteredAndLoggedIn(t *testing.T, conn *sql.DB, response *httptest.ResponseRecorder, fields map[string]string) {
	t.Helper()
	var session *http.Cookie
	for _, c := range response.Result().Cookies() {
		if c.Name == "session" {
			session = c
		}
	}
	if session == nil {
		t.Fatal("no session cookie after registration")
	}
	login := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	login.AddCookie(session)
	if user := currentUser(login); user == nil || user.Username != fields["username"] {
		t.Errorf("session cookie gives user %#v, want %q", user, fields["username"])
	}

	var passwordHash string
	if err := conn.QueryRowContext(t.Context(), "SELECT password FROM users WHERE username = ?", fields["username"]).Scan(&passwordHash); err != nil {
		t.Fatalf("query registered user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(fields["password"])); err != nil {
		t.Errorf("stored password does not match: %v", err)
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
