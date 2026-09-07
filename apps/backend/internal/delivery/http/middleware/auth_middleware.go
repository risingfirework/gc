package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tka/apps/backend/internal/domain"
)

type claimsContextKey struct{}

func RequireAuth(auth domain.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := strings.TrimSpace(r.Header.Get("Authorization"))
			parts := strings.Fields(header)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
				writeUnauthorized(w)
				return
			}

			claims, err := auth.ValidateAccessToken(r.Context(), parts[1])
			if err != nil {
				if errors.Is(err, domain.ErrSessionInvalid) {
					writeUnauthorized(w)
					return
				}
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authentication service unavailable"})
				return
			}

			ctx := context.WithValue(r.Context(), claimsContextKey{}, *claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClaimsFromContext(ctx context.Context) (domain.AuthClaims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(domain.AuthClaims)
	return claims, ok
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok || claims.Role != domain.RoleAdmin {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin access required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireTeacher(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok || (claims.Role != domain.RoleTeacher && claims.Role != domain.RoleAdmin) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "teacher access required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeUnauthorized(w http.ResponseWriter) {
	writeJSON(w, http.StatusUnauthorized, map[string]string{
		"error": "Device Limit Reached / Session Invalid",
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
