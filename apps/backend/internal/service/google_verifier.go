package service

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"tka/apps/backend/internal/domain"
)

const googleJWKSURL = "https://www.googleapis.com/oauth2/v3/certs"

var googleIssuers = map[string]bool{"accounts.google.com": true, "https://accounts.google.com": true}

// GoogleVerifier validates Google Sign-In ID tokens (RS256) against Google's published JWKS.
type GoogleVerifier struct {
	clientID   string
	httpClient *http.Client

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

func NewGoogleVerifier(clientID string) *GoogleVerifier {
	return &GoogleVerifier{clientID: clientID, httpClient: &http.Client{Timeout: 5 * time.Second}}
}

type googleClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	jwt.RegisteredClaims
}

func (v *GoogleVerifier) Verify(ctx context.Context, idToken string) (*domain.GoogleIdentity, error) {
	claims := &googleClaims{}
	token, err := jwt.ParseWithClaims(idToken, claims, func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, fmt.Errorf("google id token missing kid header")
		}
		return v.key(ctx, kid)
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}), jwt.WithAudience(v.clientID), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid google id token: %w", err)
	}
	if !googleIssuers[claims.Issuer] || claims.Email == "" {
		return nil, fmt.Errorf("invalid google id token claims")
	}
	return &domain.GoogleIdentity{Email: claims.Email, EmailVerified: claims.EmailVerified}, nil
}

func (v *GoogleVerifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if key, ok := v.keys[kid]; ok && time.Since(v.fetchedAt) < time.Hour {
		return key, nil
	}
	keys, err := v.fetchKeys(ctx)
	if err != nil {
		return nil, err
	}
	v.keys, v.fetchedAt = keys, time.Now()
	key, ok := keys[kid]
	if !ok {
		return nil, fmt.Errorf("unknown google signing key %q", kid)
	}
	return key, nil
}

type googleJWK struct {
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (v *GoogleVerifier) fetchKeys(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, googleJWKSURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := v.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch google jwks: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch google jwks: unexpected status %d", response.StatusCode)
	}
	var set struct {
		Keys []googleJWK `json:"keys"`
	}
	if err := json.NewDecoder(response.Body).Decode(&set); err != nil {
		return nil, fmt.Errorf("decode google jwks: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, key := range set.Keys {
		publicKey, err := parseGoogleRSAKey(key.N, key.E)
		if err != nil {
			continue
		}
		keys[key.Kid] = publicKey
	}
	return keys, nil
}

func parseGoogleRSAKey(modulus, exponent string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(modulus)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(exponent)
	if err != nil {
		return nil, err
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: int(new(big.Int).SetBytes(eBytes).Int64())}, nil
}

var _ domain.GoogleIDTokenVerifier = (*GoogleVerifier)(nil)
