// Package middleware menyediakan middleware HTTP (auth, rate limit, keamanan).
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

func RequireRoles(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "staff access required"})
				return
			}
			if _, ok := allowed[claims.Role]; !ok {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "staff access required"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireAdmin(next http.Handler) http.Handler {
	// Admin (operator konten) dan owner (pemilik) diizinkan; finance tidak.
	return RequireRoles(domain.RoleOwner, domain.RoleAdmin)(next)
}

func RequireTeacher(next http.Handler) http.Handler {
	return RequireRoles(domain.RoleTeacher, domain.RoleAdmin, domain.RoleOwner)(next)
}

// RequireAffiliate membatasi akses hanya untuk user dengan role affiliate.
func RequireAffiliate(next http.Handler) http.Handler {
	return RequireRoles(domain.RoleAffiliate)(next)
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
