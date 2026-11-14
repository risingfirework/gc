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
				// Fail-closed: bila penyimpanan rate limit tidak tersedia,
				// endpoint autentikasi tidak boleh berjalan tanpa proteksi.
				if logger != nil {
					logger.Error("rate limiter unavailable; rejecting request", "error", err)
				}
				w.Header().Set("Retry-After", "30")
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{
					"error": "Layanan sedang sibuk. Coba lagi sebentar lagi.",
				})
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
// Saat trustProxy aktif, X-Real-IP dari nginx diprioritaskan; entri terakhir
// X-Forwarded-For dipakai sebagai fallback karena nginx menambahkan IP klien
// asli di ujung rantai.
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
		if real := strings.TrimSpace(r.Header.Get("X-Real-IP")); net.ParseIP(real) != nil {
			return real
		}
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			parts := strings.Split(forwarded, ",")
			for i := len(parts) - 1; i >= 0; i-- {
				candidate := strings.TrimSpace(parts[i])
				if net.ParseIP(candidate) != nil {
					return candidate
				}
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}
