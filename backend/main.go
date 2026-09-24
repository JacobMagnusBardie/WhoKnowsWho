package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/securecookie"
	"github.com/joho/godotenv"

	httpSwagger "github.com/swaggo/http-swagger" // swagger UI handler
	"golang.org/x/crypto/bcrypt"
	_ "whoknows/API-specs" // head -1 go.mod (module path) + /API-specs
)

// Parses all html pages through Go's template engine. The templates are stored in the "templates" variable and can be used to render HTML pages with dynamic data.
const htmlDir = "../frontend/html/"
const contentTypeHTML = "text/html; charset=utf-8"

// Each page pairs the shared layout with its own body file, so layout.html template knows what .html to render with a layout. (See L. 25 layout.html)
var pages = map[string]*template.Template{
	"login":  template.Must(template.ParseFiles(htmlDir+"layout.html", htmlDir+"login.html")),
	"search": template.Must(template.ParseFiles(htmlDir+"layout.html", htmlDir+"search.html")),
}

// sessionKey signs/verifies session cookie values. Initialized in main() from SESSION_HASH_KEY.
var sessionKey *securecookie.SecureCookie

// cookieSecure marks the session cookie as HTTPS-only. Off by default because the VM serves
// plain HTTP, where browsers and the simulator would drop a Secure cookie. Set COOKIE_SECURE=true
// once HTTPS is in place.
var cookieSecure bool

// loadSessionKey reads SESSION_HASH_KEY (a base64-encoded 32-byte value) from the environment
// and builds the SecureCookie codec used to sign session cookies.
func loadSessionKey() *securecookie.SecureCookie {
	encoded := os.Getenv("SESSION_HASH_KEY")
	if encoded == "" {
		log.Fatal("SESSION_HASH_KEY is not set")
	}

	hashKey, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		log.Fatalf("SESSION_HASH_KEY is not valid base64: %v", err)
	}

	return securecookie.New(hashKey, nil)
}

// currentUser reads the session cookie and returns the logged-in user, or nil if there
// is no valid session.
func currentUser(r *http.Request) *User {
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil
	}

	values := map[string]interface{}{}
	if err := sessionKey.Decode("session", cookie.Value, &values); err != nil {
		return nil
	}

	username, ok := values["username"].(string)
	if !ok {
		return nil
	}

	return &User{Username: username}
}

// @Summary Serve Root Page
// @Param q query string false "Search query"
// @Router / [get]
func serveRootPage(w http.ResponseWriter, r *http.Request) {
	// The search page is public (OpenAPI spec: GET / returns 200 text/html); the session only
	// decides whether the nav shows "Log out" or "Log in / Register".
	w.Header().Set("Content-Type", contentTypeHTML)
	query := r.URL.Query().Get("q") // Get the value of the "q" query parameter from the URL. If the parameter is not present, query will be an empty string.

	// TODO: erstat med rigtigt DB-opslag mod pages-tabellen
	results := []SearchResult{} //Array of SearchResult structs, which is empty for now. This will be populated with search results from the database in the future.

	if err := pages["search"].ExecuteTemplate(w, "layout", PageData{Title: "¿Who Knows?", Query: query, Results: results, User: currentUser(r)}); err != nil {
		log.Printf("render search page: %v", err)
	}
}

// @Summary Serve Register Page
// @Router /register [get]
func serveRegisterPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeHTML)
	fmt.Fprintln(w, "<h1>Register</h1>")
}

// @Summary Serve Login Page
// @Router /login [get]
func serveLoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeHTML)
	// Render the login.html template. The PageData struct is populated with the title "Log In" and passed to the template (layout.html) and then to login.html for rendering. (see l. 2 in layout.html)
	if err := pages["login"].ExecuteTemplate(w, "layout", PageData{Title: "Log In", User: currentUser(r)}); err != nil {
		log.Printf("render login page: %v", err)
	}
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

	user, err := getUserByUsername(username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// user is nil when the username doesn't exist. Answer exactly like a wrong password,
	// so the response doesn't reveal which usernames are registered.
        if user == nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		w.WriteHeader(http.StatusUnauthorized)
		statusCode := 401
		message := "Invalid username or password"
		json.NewEncoder(w).Encode(AuthResponse{StatusCode: &statusCode, Message: &message})
		return
	}

	// Create a session cookie for the logged-in user. The cookie is signed using the sessionKey to prevent tampering. The cookie contains the user's ID, which can be used to identify the user in subsequent requests.
	// The cookie is set to HttpOnly to prevent access from JavaScript, and SameSite is set to Lax to allow the cookie to be sent with top-level navigations.
	// The cookie is set to expire when the browser session ends (no MaxAge or Expires is set).
	// (use gorilla/sessions for more advanced session management, e.g. with Redis or database-backed sessions)
	encoded, err := sessionKey.Encode("session", map[string]interface{}{"user_id": user.ID, "username": user.Username})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // G124: Secure follows COOKIE_SECURE, the VM serves plain HTTP
		Name:     "session",
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		Secure:   cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	statusCode := 200
	message := "Logged in successfully"
	json.NewEncoder(w).Encode(AuthResponse{StatusCode: &statusCode, Message: &message})
}

// @Summary Logout
// @Success 200 {object} AuthResponse
// @Router /api/logout [get]
func apiLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // G124: Secure follows COOKIE_SECURE, the VM serves plain HTTP
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1, // MaxAge -1 means delete the cookie immediately
	})

	// OpenAPI spec: 200 application/json with an AuthResponse, not a redirect
	statusCode := 200
	message := "Logged out successfully"
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(AuthResponse{StatusCode: &statusCode, Message: &message}); err != nil {
		log.Printf("writing logout response: %v", err)
	}
}

// @title WhoKnows API
// @version 0.1.0
// @description API for søgning, login og vejr i WhoKnows-projektet.
// @host localhost:8080
// @BasePath /
func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on process environment")
	}
	sessionKey = loadSessionKey()
	cookieSecure = os.Getenv("COOKIE_SECURE") == "true"

	// Initialize the database connection and ensure the schema is applied.
	// The db variable is a global handle to the SQLite database, which is used by the API handlers to perform queries and updates.
	db = initDB()
	defer db.Close() // defer meaning: ensures that the database connection is closed when the main function exits, preventing resource leaks.

	mux := http.NewServeMux()

	// Serve /static/* (CSS, images, osv) from ../frontend/static
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("../frontend/static"))))

	mux.HandleFunc("GET /", serveRootPage)
	mux.HandleFunc("GET /weather", serveWeatherPage)
	mux.HandleFunc("GET /register", serveRegisterPage)
	mux.HandleFunc("GET /login", serveLoginPage)

	mux.HandleFunc("GET /api/search", apiSearch)
	mux.HandleFunc("GET /api/weather", apiWeatherHandler)
	mux.HandleFunc("POST /api/register", apiRegister)
	mux.HandleFunc("POST /api/login", apiLogin)
	mux.HandleFunc("GET /api/logout", apiLogout)

	// Swagger UI endpoint
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	fmt.Println("Server running on :8080")
	http.ListenAndServe(":8080", mux)
}
