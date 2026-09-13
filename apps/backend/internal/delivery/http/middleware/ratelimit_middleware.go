package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tka/apps/backend/internal/domain"
)

// RateLimit membatasi pemakaian per identitas client pada sebuah rute
// dengan jendela waktu tetap. Untuk endpoint /login, kunci dibangun
// dari email (per-user) sehingga semua request dari IP proxy yang sama
// tidak saling menghabiskan kuota. Endpoint lain tetap per-IP.
// Saat Redis bermasalah, request dilewatkan (fail-open).
// trustProxy=false (default) membuat X-Forwarded-For diabaikan agar
// header palsu dari client tidak bisa mengelabui limiter; saat dipasang
// di belakang nginx yang sudah dipetakan, set TRUST_PROXY=true.
func RateLimit(limiter domain.RateLimiter, limit int, window time.Duration, logger *slog.Logger, trustProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key, body := rateLimitKey(r, trustProxy)
			// Refresh reader agar downstream bisa membaca body lagi.
			if body != nil {
				r.Body = io.NopCloser(bytes.NewReader(body))
			}
			allowed, err := limiter.Allow(r.Context(), key, limit, window)
			if err != nil {
				if logger != nil {
					logger.Warn("rate limiter unavailable; allowing request", "error", err)
				}
				next.ServeHTTP(w, r)
				return
			}
			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(int(window.Seconds())))
				writeJSON(w, http.StatusTooManyRequests, map[string]string{
					"error": "Terlalu banyak percobaan. Tunggu sebentar lalu coba lagi.",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// rateLimitKey mengembalikan kunci rate limit. Untuk /login, pakai email
// dari body JSON agar per-user. Untuk rute lain, pakai IP + rute.
func rateLimitKey(r *http.Request, trustProxy bool) (string, []byte) {
	// Salin body agar bisa dibaca ulang.
	raw, _ := io.ReadAll(r.Body)
	email := ""

	if strings.HasSuffix(r.URL.Path, "/login") && len(raw) > 0 {
		var parsed struct {
			Email string `json:"email"`
		}
		if json.Unmarshal(raw, &parsed) == nil && parsed.Email != "" {
			email = strings.ToLower(strings.TrimSpace(parsed.Email))
		}
	}

	if email != "" {
		return "email:" + email + ":" + r.URL.Path, raw
	}
	return "ip:" + clientIP(r, trustProxy) + ":" + r.URL.Path, raw
}

func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			first := strings.TrimSpace(strings.Split(forwarded, ",")[0])
			if net.ParseIP(first) != nil {
				return first
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}
