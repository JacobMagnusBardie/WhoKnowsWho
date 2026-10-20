package main

type AuthResponse struct {
	StatusCode *int    `json:"statusCode,omitempty"`
	Message    *string `json:"message,omitempty"`
} // @name AuthResponse

type SearchResponse struct {
	Data []SearchResult `json:"data"`
} // @name SearchResponse

type StandardResponse struct {
	Data map[string]interface{} `json:"data"`
} // @name StandardResponse

type ValidationError struct {
	Loc  []interface{} `json:"loc"`
	Msg  string        `json:"msg"`
	Type string        `json:"type"`
} // @name ValidationError

type HTTPValidationError struct {
	Detail []ValidationError `json:"detail"`
} // @name HTTPValidationError

type User struct {
	ID       int
	Username string
	Password string
}

// Credentials holds the fields posted to /api/register and /api/login, whether they
// arrive as JSON or form-encoded. Email and Password2 are only used by register.
type Credentials struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Password2 string `json:"password2"`
}

// WeatherInfo holds the forecast values shown on the weather page.
// Kept as its own small struct (rather than loose fields directly on PageData)
// so the template can simply check `{{if .Weather}}` to know whether forecast
// data is available at all.
type WeatherInfo struct {
	Temperature float64
	WindSpeed   float64
	Humidity    float64
}

type PageData struct {
	Title    string
	Error    string
	Username string
	Email    string
	User     *User // nil if logged out
	Flashes  []string
	Weather  *WeatherInfo // nil unless the page is rendering a forecast
	Query    string
	Results  []SearchResult
}

type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
} // @name SearchResult
