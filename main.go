package main

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)


// helper to convert environment string to slog.Level
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info", "information":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}


// look in the request headers for a proxy IP
func getProxyIP(r *http.Request) string {
    xff := r.Header.Get("X-Forwarded-For")
    if xff != "" {
        for ip := range strings.SplitSeq(xff, ",") {
            return ip
        }
    }

    // check Forwarded header (RFC 7239) as fallback
    if forwarded := r.Header.Get("Forwarded"); forwarded != "" {
        // forwarded format: "for=client;by=proxy"
        for part := range strings.SplitSeq(forwarded, ";") {
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


func rateLimitThis(r *http.Request, whitelistIP string, trustedProxy string, logger *slog.Logger) bool {
    clientIP := r.RemoteAddr
    proxyIP := getProxyIP(r)
    if logger != nil {
        logger.Debug("Rate limit check", "clientIP", clientIP, "proxyIP", proxyIP, "whitelistIP", whitelistIP, "trustedProxy", trustedProxy)
    }
    
    if whitelistIP == "" {
        // there's no whitelisted ip, rate limit everything
        return true
    }

    if clientIP == whitelistIP {
        // request is directly from whitelisted IP, no rate limit
        return false
    }

    if (clientIP == trustedProxy) && (trustedProxy != "") {
        // request is from the trusted proxy, don't rate limit if whitelist IP is in headers
        return !(proxyIP == whitelistIP)
    }

    // there's no trusted proxy, or this request is not from it
    return true  
}


func main() {
    // set up logger
	logLevel := os.Getenv("LOG_LEVEL")
	level := parseLogLevel(logLevel)
	opts := &slog.HandlerOptions{Level: level}
	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))

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
        logger.Warn("Invalid RATE_LIMIT_MS value, using default 1000ms")
        rateLimitDuration = 1000
    }
    
    whitelistIP := os.Getenv("RATE_LIMIT_WHITELIST_IP")
    trustedProxyIP := os.Getenv("TRUSTED_PROXY_IP")

    var (
        lastRequest time.Time
        mu          sync.Mutex
    )

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        
        if rateLimitThis(r, whitelistIP, trustedProxyIP, logger) {
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
        
        w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
        w.WriteHeader(resp.StatusCode)
        io.Copy(w, resp.Body)
    })

    logger.Info("mini-proxy starting", "port", port, "target", targetURL)
    logger.Info("rate limit configured", "duration_ms", rateLimitDuration)
    if whitelistIP != "" {
        logger.Info("whitelist configured", "ip", whitelistIP)
    }
    if trustedProxyIP != "" {
        logger.Info("trusted proxy configured", "ip", trustedProxyIP)
    }
    err = http.ListenAndServe(":"+port, nil)
    if err != nil {
        logger.Error("server failed to start", "error", err)
        os.Exit(1)
    }
}
