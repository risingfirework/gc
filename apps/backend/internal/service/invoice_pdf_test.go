package service

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"tka/apps/backend/internal/domain"
)

func TestRenderInvoicePDF(t *testing.T) {
	paidAt := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	invoice := domain.Invoice{
		PlatformName:    "TKA",
		PlatformTagline: "Belajar seru, ranking raih.",
		InvoiceNumber:   "TKA-20260908-120000-ABCDEF12",
		BuyerName:       "Siswa SMA 1",
		BuyerEmail:      "siswa.sma1@tka.local",
		BuyerSchool:     "SMA",
		PackageTitle:    "Bundle Matematika (ujian+soal)",
		PackageKode:     "MAT-BUNDLE-01",
		PackageJenjang:  "SMA",
		PackageValidity: 30,
		Price:           25000,
		PaymentMethod:   "qris",
		PaymentStatus:   "paid",
		PaidAt:          &paidAt,
		CreatedAt:       paidAt,
		TransactionID:   "4fe6f1fc-50d0-4ac6-9df6-686db9ffbcf2",
	}
	pdf := RenderInvoicePDF(invoice)
	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) {
		t.Fatalf("pdf is missing header: %q", pdf[:min(80, len(pdf))])
	}
	if !strings.Contains(string(pdf), "%%EOF") {
		t.Fatal("pdf is missing EOF marker")
	}
	if !strings.Contains(string(pdf), "TKA-20260908-120000-ABCDEF12") {
		t.Fatal("pdf does not embed the invoice number")
	}
	if !strings.Contains(string(pdf), "Rp 25.000") {
		t.Fatal("pdf does not embed the money total")
	}
	// Referensi objek wajib memakai sintaks "N 0 R", bukan "N 0 obj",
	// agar pdf valid dan bisa dibuka validator/reader.
	if strings.Contains(string(pdf), " 0 obj >>") || strings.Contains(string(pdf), "/Root 6 0 obj") {
		for i, line := range strings.Split(string(pdf), "\n") {
			if strings.Contains(line, "0 obj") && i > 0 {
				t.Fatalf("pdf contains invalid object reference: %q", line)
			}
		}
	}
	if !strings.Contains(string(pdf), "/Parent 5 0 R") {
		t.Fatal("pdf page is missing a valid /Parent reference")
	}
	if !strings.Contains(string(pdf), "/Root 6 0 R") {
		t.Fatal("pdf trailer is missing a valid /Root reference")
	}
}
