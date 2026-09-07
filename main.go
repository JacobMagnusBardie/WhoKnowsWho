package main

  import (
        "encoding/json"
        "log"
        "net/http"
  )

  func routes() *http.ServeMux {
        mux := http.NewServeMux()

        // GET /api/weather -> StandardResponse. Stub: the assignment exempts it.
        mux.HandleFunc("GET /api/weather", func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
        })

        return mux
  }

  func main() {
        log.Println("listening on http://localhost:8081")
        log.Fatal(http.ListenAndServe(":8081", routes()))
  }