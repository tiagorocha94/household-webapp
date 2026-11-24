package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsCrossSite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		headers map[string]string
		want    bool
	}{
		{"no headers, like curl", nil, false},
		{"same origin", map[string]string{"Sec-Fetch-Site": "same-origin"}, false},
		{"typed address or bookmark", map[string]string{"Sec-Fetch-Site": "none"}, false},
		{"another site", map[string]string{"Sec-Fetch-Site": "cross-site"}, true},
		{"another port on the same site", map[string]string{"Sec-Fetch-Site": "same-site"}, true},
		{"fetch metadata wins over origin", map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "http://example.com"}, true},
		{"matching origin", map[string]string{"Origin": "http://example.com"}, false},
		{"other origin", map[string]string{"Origin": "http://evil.example"}, true},
		{"origin on another port", map[string]string{"Origin": "http://example.com:3000"}, true},
		{"origin that is not a URL", map[string]string{"Origin": "://"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/auth/logout", nil)
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}
			if got := isCrossSite(r); got != tt.want {
				t.Errorf("isCrossSite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientIPIgnoresThePortAndForwardingHeaders(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = "203.0.113.7:51234"
	r.Header.Set("X-Forwarded-For", "198.51.100.1")
	if got := clientIP(r); got != "203.0.113.7" {
		t.Errorf("clientIP() = %q, want 203.0.113.7", got)
	}

	r.RemoteAddr = "no-port"
	if got := clientIP(r); got != "no-port" {
		t.Errorf("clientIP() = %q, want the raw address when it has no port", got)
	}
}
