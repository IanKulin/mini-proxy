package main

import (
	"net/http"
	"testing"
)

func TestGetProxyIP(t *testing.T) {
	tests := []struct {
		name           string
		headers        map[string]string
		expectedIP     string
	}{
		{
			name:       "X-Forwarded-For single IP",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.100"},
			expectedIP: "192.168.1.100",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.100, 10.0.0.1, 172.16.0.1"},
			expectedIP: "192.168.1.100",
		},
		{
			name:       "Forwarded header with IPv4",
			headers:    map[string]string{"Forwarded": "for=192.168.1.100;by=proxy"},
			expectedIP: "192.168.1.100",
		},
		{
			name:       "Forwarded header with IPv4 and port",
			headers:    map[string]string{"Forwarded": "for=192.168.1.100:8080;by=proxy"},
			expectedIP: "192.168.1.100",
		},
		{
			name:       "Forwarded header with IPv6",
			headers:    map[string]string{"Forwarded": "for=\"[2001:db8::1]\";by=proxy"},
			expectedIP: "2001:db8::1",
		},
		{
			name:       "Forwarded header with quoted IPv4",
			headers:    map[string]string{"Forwarded": "for=\"192.168.1.100\";by=proxy"},
			expectedIP: "192.168.1.100",
		},
		{
			name:       "No proxy headers",
			headers:    map[string]string{},
			expectedIP: "",
		},
		{
			name:       "Both headers present (X-Forwarded-For takes precedence)",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.100", "Forwarded": "for=10.0.0.1;by=proxy"},
			expectedIP: "192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			result := getProxyIP(req)
			if result != tt.expectedIP {
				t.Errorf("getProxyIP() = %v, want %v", result, tt.expectedIP)
			}
		})
	}
}

func TestRateLimitThis(t *testing.T) {
	tests := []struct {
		name             string
		remoteAddr       string
		headers          map[string]string
		whitelistIP      string
		onlyTrustedProxy string
		expected         bool
	}{
		{
			name:             "No whitelist IP - should rate limit",
			remoteAddr:       "192.168.1.100",
			whitelistIP:      "",
			onlyTrustedProxy: "",
			expected:         true,
		},
		{
			name:             "Direct connection from whitelisted IP - no rate limit",
			remoteAddr:       "192.168.1.100",
			whitelistIP:      "192.168.1.100",
			onlyTrustedProxy: "",
			expected:         false,
		},
		{
			name:             "Direct connection from non-whitelisted IP - rate limit",
			remoteAddr:       "192.168.1.200",
			whitelistIP:      "192.168.1.100",
			onlyTrustedProxy: "",
			expected:         true,
		},
		{
			name:             "Untrusted proxy mode - whitelisted IP in headers - rate limit (headers ignored)",
			remoteAddr:       "10.0.0.1",
			headers:          map[string]string{"X-Forwarded-For": "192.168.1.100"},
			whitelistIP:      "192.168.1.100",
			onlyTrustedProxy: "",
			expected:         true,
		},
		{
			name:             "Untrusted proxy mode - non-whitelisted IP in headers - rate limit",
			remoteAddr:       "10.0.0.1",
			headers:          map[string]string{"X-Forwarded-For": "192.168.1.200"},
			whitelistIP:      "192.168.1.100",
			onlyTrustedProxy: "",
			expected:         true,
		},
		{
			name:             "Trusted proxy mode - request from trusted proxy with whitelisted IP in headers - no rate limit",
			remoteAddr:       "10.0.0.1",
			headers:          map[string]string{"X-Forwarded-For": "192.168.1.100"},
			whitelistIP:      "192.168.1.100",
			onlyTrustedProxy: "10.0.0.1",
			expected:         false,
		},
		{
			name:             "Trusted proxy mode - request from trusted proxy with non-whitelisted IP in headers - rate limit",
			remoteAddr:       "10.0.0.1",
			headers:          map[string]string{"X-Forwarded-For": "192.168.1.200"},
			whitelistIP:      "192.168.1.100",
			onlyTrustedProxy: "10.0.0.1",
			expected:         true,
		},
		{
			name:             "Trusted proxy mode - request from untrusted proxy - rate limit",
			remoteAddr:       "10.0.0.2",
			headers:          map[string]string{"X-Forwarded-For": "192.168.1.100"},
			whitelistIP:      "192.168.1.100",
			onlyTrustedProxy: "10.0.0.1",
			expected:         true,
		},
		{
			name:             "Trusted proxy mode - direct request from whitelisted IP bypassing trusted proxy - no rate limit",
			remoteAddr:       "192.168.1.100",
			whitelistIP:      "192.168.1.100",
			onlyTrustedProxy: "10.0.0.1",
			expected:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			result := rateLimitThis(req, tt.whitelistIP, tt.onlyTrustedProxy, nil)
			if result != tt.expected {
				t.Errorf("rateLimitThis() = %v, want %v", result, tt.expected)
			}
		})
	}
}