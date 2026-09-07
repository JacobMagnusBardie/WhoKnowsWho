package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// @Summary Serve Root Page
// @Router / [get]
func serveRootPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintln(w, "<h1>WhoKnows</h1>")
}

// @Summary Serve Weather Page
// @Router /weather [get]
func serveWeatherPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintln(w, "<h1>Weather</h1>")
}

// @Summary Serve Register Page
// @Router /register [get]
func serveRegisterPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintln(w, "<h1>Register</h1>")
}

// @Summary Serve Login Page
// @Router /login [get]
func serveLoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintln(w, "<h1>Login</h1>")
}

// @Summary Search
// @Param q query string true "Search query"
// @Param language query string false "Language code (e.g. 'en')"
// @Success 200 {object} SearchResponse
// @Failure 422 {object} HTTPValidationError
// @Router /api/search [get]
func apiSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	w.Header().Set("Content-Type", "application/json")

	if q == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(HTTPValidationError{
			Detail: []ValidationError{
				{Loc: []interface{}{"query", "q"}, Msg: "Field required", Type: "missing"},
			},
		})
		return
	}

	// TODO: erstat med rigtigt DB-opslag mod pages-tabellen
	results := []map[string]interface{}{}
	json.NewEncoder(w).Encode(SearchResponse{Data: results})
}

// @Summary Weather
// @Success 200 {object} StandardResponse
// @Router /api/weather [get]
func apiWeather(w http.ResponseWriter, r *http.Request) {
	// TODO: erstat med rigtigt vejr-data-opslag
	data := map[string]interface{}{"temperature": "unknown"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(StandardResponse{Data: data})
}

// @Summary Register
// @Param username formData string true "Username"
// @Param email formData string true "Email"
// @Param password formData string true "Password"
// @Success 200 {object} AuthResponse
// @Failure 422 {object} HTTPValidationError
// @Router /api/register [post]
func apiRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	if username == "" || email == "" || password == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(HTTPValidationError{
			Detail: []ValidationError{{Loc: []interface{}{"body"}, Msg: "Missing required field", Type: "missing"}},
		})
		return
	}

	// TODO: hash password med bcrypt, tjek om username findes, indsæt i DB

	statusCode := 200
	message := "User registered successfully"
	json.NewEncoder(w).Encode(AuthResponse{StatusCode: &statusCode, Message: &message})
}

// @Summary Login
// @Param username formData string true "Username"
// @Param password formData string true "Password"
// @Success 200 {object} AuthResponse
// @Failure 422 {object} HTTPValidationError
// @Router /api/login [post]
func apiLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(HTTPValidationError{
			Detail: []ValidationError{{Loc: []interface{}{"body"}, Msg: "Missing required field", Type: "missing"}},
		})
		return
	}

	// TODO: verificér mod DB (bcrypt.CompareHashAndPassword), opret session

	statusCode := 200
	message := "Logged in successfully"
	json.NewEncoder(w).Encode(AuthResponse{StatusCode: &statusCode, Message: &message})
}

// @Summary Logout
// @Success 200 {object} AuthResponse
// @Router /api/logout [get]
func apiLogout(w http.ResponseWriter, r *http.Request) {
	// TODO: ryd session/cookie
	statusCode := 200
	message := "Logged out successfully"
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{StatusCode: &statusCode, Message: &message})
}

// @title WhoKnows API
// @version 0.1.0
// @description API for søgning, login og vejr i WhoKnows-projektet.
// @host localhost:8080
// @BasePath /
func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", serveRootPage)
	mux.HandleFunc("GET /weather", serveWeatherPage)
	mux.HandleFunc("GET /register", serveRegisterPage)
	mux.HandleFunc("GET /login", serveLoginPage)

	mux.HandleFunc("GET /api/search", apiSearch)
	mux.HandleFunc("GET /api/weather", apiWeather)
	mux.HandleFunc("POST /api/register", apiRegister)
	mux.HandleFunc("POST /api/login", apiLogin)
	mux.HandleFunc("GET /api/logout", apiLogout)

	fmt.Println("Server running on :8080")
	http.ListenAndServe(":8080", mux)
}