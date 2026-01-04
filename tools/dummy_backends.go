package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

// A simple program to spin up 3 backend servers for testing the Load Balancer
func main() {
	ports := []string{"8081", "8082", "8083"}
	var wg sync.WaitGroup

	for _, port := range ports {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				fmt.Printf("Backend %s received request\n", p)
				fmt.Fprintf(w, "Hello from Backend :%s\n", p)
			})

			log.Printf("Starting backend on :%s\n", p)
			if err := http.ListenAndServe(":"+p, mux); err != nil {
				log.Fatal(err)
			}
		}(port)
	}

	wg.Wait()
}
