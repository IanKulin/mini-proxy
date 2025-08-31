package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)
 

func getProxyIP(r *http.Request) string {
    xff := r.Header.Get("X-Forwarded-For")
    if xff != "" {
        ips := strings.Split(xff, ",")
        if len(ips) > 0 {
            return ips[0]
        }
    }

    // Check Forwarded header (RFC 7239) as fallback
    if forwarded := r.Header.Get("Forwarded"); forwarded != "" {
        // Forwarded format: "for=client;by=proxy"
        parts := strings.Split(forwarded, ";")
        for _, part := range parts {
            part = strings.TrimSpace(part)
            if strings.HasPrefix(part, "for=") {
                forValue := strings.TrimPrefix(part, "for=")
                // Remove quotes if present
                forValue = strings.Trim(forValue, "\"")
                // Handle IPv6 brackets or IPv4 with port
                if strings.HasPrefix(forValue, "[") && strings.Contains(forValue, "]") {
                    end := strings.Index(forValue, "]")
                    forValue = forValue[1:end]
                } else if strings.Contains(forValue, ":") {
                    // IPv4 with port
                    host, _, err := net.SplitHostPort(forValue)
                    if err == nil {
                        forValue = host
                    }
                }
                return forValue
            }
        }
    }
    return ""
}


func rateLimitThis(r *http.Request, whitelistIP string, onlyTrustedProxy string) bool {
    clientIP := r.RemoteAddr
    proxyIP := getProxyIP(r)

    if whitelistIP == "" {
        // if there's no whitelisted ip, rate limit everything
        return true
    }

    if onlyTrustedProxy != "" {
        // if there's a trusted proxy defined, only trust that proxy
        if clientIP == onlyTrustedProxy {
            // request is from trusted proxy, check if whitelist IP is in headers
            return !(proxyIP == whitelistIP)
        } else {
            // request is not from trusted proxy, rate limit
            return true
        }
    } else {
        // if there's no trusted proxy defined, we trust all proxies
        // check both RemoteAddr and headers for whitelist IP
        return !((clientIP == whitelistIP) || (proxyIP == whitelistIP))
    }   
}


func main() {
    targetURL := os.Getenv("TARGET_HEALTH_URL")
    if targetURL == "" {
        targetURL = "http://your-app:8080/health" // default
    }
    
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    rateLimitMs := os.Getenv("RATE_LIMIT_MS")
    if rateLimitMs == "" {
        rateLimitMs = "1000"
    }
    
    rateLimitDuration, err := strconv.Atoi(rateLimitMs)
    if err != nil {
        log.Printf("Invalid RATE_LIMIT_MS value, using default 1000ms")
        rateLimitDuration = 1000
    }
    
    whitelistIP := os.Getenv("RATE_LIMIT_WHITELIST_IP")
    onlyTrustProxyIP := os.Getenv("ONLY_TRUST_PROXY_IP")

    var (
        lastRequest time.Time
        mu          sync.Mutex
    )

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        
        if rateLimitThis(r, whitelistIP, onlyTrustProxyIP) {
            mu.Lock()
            now := time.Now()
            if !lastRequest.IsZero() && now.Sub(lastRequest) < time.Duration(rateLimitDuration)*time.Millisecond {
                mu.Unlock()
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }
            lastRequest = now
            mu.Unlock()
        }

        resp, err := http.Get(targetURL)
        if err != nil {
            http.Error(w, "Proxy failed", http.StatusServiceUnavailable)
            return
        }
        defer resp.Body.Close()
        
        w.WriteHeader(resp.StatusCode)
        w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
        io.Copy(w, resp.Body)
    })

    log.Printf("mini-proxy listening on :%s, proxying to %s", port, targetURL)
    log.Printf("Rate limit: %dms between requests", rateLimitDuration)
    if whitelistIP != "" {
        log.Printf("Whitelist IP: %s", whitelistIP)
    }
    if onlyTrustProxyIP != "" {
        log.Printf("Trusted proxy: %s", onlyTrustProxyIP)
    }
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
