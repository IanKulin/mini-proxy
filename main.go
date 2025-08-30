package main

import (
    "io"
    "log"
    "net/http"
    "os"
)

func main() {
    targetURL := os.Getenv("TARGET_HEALTH_URL")
    if targetURL == "" {
        targetURL = "http://your-app:8080/health" // default
    }
    
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        resp, err := http.Get(targetURL)
        if err != nil {
            http.Error(w, "Health check failed", http.StatusServiceUnavailable)
            return
        }
        defer resp.Body.Close()
        
        w.WriteHeader(resp.StatusCode)
        w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
        io.Copy(w, resp.Body)
    })

    log.Printf("Health proxy listening on :%s, proxying to %s", port, targetURL)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}