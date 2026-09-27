package ws

import (
	"net/http/httptest"
	"testing"
)

func TestCheckOrigin(t *testing.T) {
	check := checkOrigin([]string{"http://localhost:5173", "https://app.exemplo.com"})

	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{"origem listada", "http://localhost:5173", true},
		{"outra origem listada", "https://app.exemplo.com", true},
		{"maiúsculas no domínio", "https://APP.exemplo.com", true},
		{"site malicioso", "https://evil.com", false},
		{"mesma origem, outra porta", "http://localhost:3000", false},
		{"http em vez de https", "http://app.exemplo.com", false},
		{"sem Origin (cliente não-navegador)", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/documents/1/ws", nil)
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			if got := check(r); got != tt.want {
				t.Errorf("Origin %q: got %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}

func TestCheckOriginEmptyListBlocksBrowsers(t *testing.T) {
	check := checkOrigin(nil)

	r := httptest.NewRequest("GET", "/documents/1/ws", nil)
	r.Header.Set("Origin", "http://localhost:5173")
	if check(r) {
		t.Error("sem ALLOWED_ORIGINS, nenhum navegador deveria conectar")
	}
}
