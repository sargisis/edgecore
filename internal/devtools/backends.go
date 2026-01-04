package devtools

import (
	"fmt"
	"log"
	"net/http"
)

// StartBackend starts a single backend server on the given port
func StartBackend(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from Backend :%s\n", port)
	})

	log.Printf("🟢 Dev Backend started on :%s\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Printf("Backend :%s failed: %v", port, err)
	}
}

// StartAllBackends starts all 3 dummy backends in goroutines
func StartAllBackends() {
	ports := []string{"8081", "8082", "8083"}
	for _, port := range ports {
		go StartBackend(port)
	}
}
