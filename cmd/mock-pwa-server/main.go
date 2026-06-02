package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	port := "8080"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	// Find the directory containing the mock PWA
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting working directory: %v", err)
	}

	pwaDir := filepath.Join(wd, "mock-pwa")
	if _, err := os.Stat(pwaDir); os.IsNotExist(err) {
		// Try parent directory
		pwaDir = filepath.Join(wd, "..", "mock-pwa")
	}

	fileServer := http.FileServer(http.Dir(pwaDir))
	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Set headers to prevent caching for dev/testing ease
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		
		log.Printf("[Server] %s %s", r.Method, r.URL.Path)
		fileServer.ServeHTTP(w, r)
	})

	fmt.Printf("\n============================================\n")
	fmt.Printf("🎵 Spotify PWA Mock Server Started!\n")
	fmt.Printf("🌍 URL: http://localhost:%s\n", port)
	fmt.Printf("📂 Serving files from: %s\n", pwaDir)
	fmt.Printf("============================================\n\n")

	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
