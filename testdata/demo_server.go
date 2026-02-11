// Minimal test server for demos. Serves a user endpoint.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func main() {
	port := "9876"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	http.HandleFunc("/users/123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    42,
			"name":  "Alice Smith",
			"email": "alice@example.com",
		})
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	fmt.Fprintf(os.Stderr, "Listening on :%s\n", port)
	http.ListenAndServe(":"+port, nil)
}
