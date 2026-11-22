package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type recordingLimiter struct {
	keys []string
}

func (l *recordingLimiter) Allow(_ context.Context, key string, _ int, _ time.Duration) (bool, error) {
	l.keys = append(l.keys, key)
	return true, nil
}

func TestRateLimitKeyRefreshUsesTokenNotIP(t *testing.T) {
	first := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`{}`))
	first.RemoteAddr = "10.0.0.5:1234"
	first.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "token-a"})

	second := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`{}`))
	second.RemoteAddr = "10.0.0.5:1234"
	second.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "token-b"})

	keyA, _ := rateLimitKey(first, false)
	keyB, _ := rateLimitKey(second, false)

	if keyA == keyB {
		t.Fatalf("different refresh tokens from the same IP must not share a key: %q", keyA)
	}
	if !strings.HasPrefix(keyA, "refresh:") || !strings.HasPrefix(keyB, "refresh:") {
		t.Fatalf("refresh keys should be token based, got %q and %q", keyA, keyB)
	}
	if strings.Contains(keyA, "token-a") || strings.Contains(keyB, "token-b") {
		t.Fatal("raw refresh token must not appear in the rate limit key")
	}
}

func TestRateLimitKeyRefreshIsStableForSameToken(t *testing.T) {
	keyFor := func(remote, body, cookie string) string {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(body))
		request.RemoteAddr = remote
		if cookie != "" {
			request.AddCookie(&http.Cookie{Name: refreshCookieName, Value: cookie})
		}
		key, _ := rateLimitKey(request, false)
		return key
	}

	fromCookie := keyFor("10.0.0.5:1234", `{}`, "same-token")
	fromBody := keyFor("10.0.0.9:4321", `{"refresh_token":"same-token"}`, "")
	if fromCookie != fromBody {
		t.Fatalf("same token should produce the same key regardless of source/IP: %q vs %q", fromCookie, fromBody)
	}
}

func TestRateLimitKeyRefreshFallsBackToIPWithoutToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`{}`))
	request.RemoteAddr = "10.0.0.5:1234"

	key, _ := rateLimitKey(request, false)
	if !strings.HasPrefix(key, "ip:10.0.0.5:") {
		t.Fatalf("expected IP fallback, got %q", key)
	}
}

func TestRateLimitKeyLoginStillUsesEmail(t *testing.T) {
	for _, path := range []string{"/api/v1/auth/login", "/api/v1/auth/register", "/api/v1/auth/forgot-password"} {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"email":"Student@Example.com"}`))
		request.RemoteAddr = "10.0.0.5:1234"

		key, _ := rateLimitKey(request, false)
		if key != "email:student@example.com:"+path {
			t.Fatalf("unexpected key for %s: %q", path, key)
		}
	}
}

func TestRateLimitRestoresBodyForDownstream(t *testing.T) {
	limiter := &recordingLimiter{}
	handler := RateLimit(limiter, 10, time.Minute, nil, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"refresh_token":"abc"}` {
			t.Errorf("downstream body was not restored: %q", body)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`{"refresh_token":"abc"}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected downstream handler to run, got status %d", recorder.Code)
	}
	if len(limiter.keys) != 1 || !strings.HasPrefix(limiter.keys[0], "refresh:") {
		t.Fatalf("expected one refresh key, got %v", limiter.keys)
	}
}
