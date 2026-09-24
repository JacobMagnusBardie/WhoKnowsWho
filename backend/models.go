package main

type AuthResponse struct {
	StatusCode *int    `json:"statusCode,omitempty"`
	Message    *string `json:"message,omitempty"`
}

type SearchResponse struct {
	Data []map[string]interface{} `json:"data"`
}

type StandardResponse struct {
	Data map[string]interface{} `json:"data"`
}

type ValidationError struct {
	Loc  []interface{} `json:"loc"`
	Msg  string        `json:"msg"`
	Type string        `json:"type"`
}

type HTTPValidationError struct {
	Detail []ValidationError `json:"detail"`
}

type User struct {
	Username string
	Password string
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
	User     *User // nil if logged out
	Flashes  []string
	Weather  *WeatherInfo // nil unless the page is rendering a forecast
}
