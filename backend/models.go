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