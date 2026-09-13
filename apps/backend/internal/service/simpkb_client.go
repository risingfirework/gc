package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// SIMPKBCheckResult adalah respons layanan simpkb-check (Playwright headless).
type SIMPKBCheckResult struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	Screenshot string `json:"screenshot"`
}

type SIMPKBClient struct {
	baseURL string
	http    *http.Client
}

// NewSIMPKBClient membuat klien HTTP untuk microservice simpkb-check.
func NewSIMPKBClient(baseURL string) *SIMPKBClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "http://simpkb-check:8080"
	}
	return &SIMPKBClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		http:    &http.Client{},
	}
}

// Check mengirim NIK ke layanan dan mengembalikan status + screenshot hasil
// pencarian di portal.simpkb.id/cari. Mengikuti timeout dari ctx.
func (c *SIMPKBClient) Check(ctx context.Context, nik string) (SIMPKBCheckResult, error) {
	body, err := json.Marshal(map[string]string{"nik": nik})
	if err != nil {
		return SIMPKBCheckResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/check", bytes.NewReader(body))
	if err != nil {
		return SIMPKBCheckResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return SIMPKBCheckResult{}, fmt.Errorf("simpkb-check service: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return SIMPKBCheckResult{}, fmt.Errorf("simpkb-check service returned %d", resp.StatusCode)
	}
	var result SIMPKBCheckResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return SIMPKBCheckResult{}, err
	}
	return result, nil
}
