package service

import (
	"fmt"
	"math"
	"strings"
	"time"

	"tka/apps/backend/internal/domain"
)

const (
	pdfPageW = 595.28
	pdfPageH = 841.89
	pdfPadL  = 45.0
	pdfPadR  = 45.0
	pdfW     = pdfPageW - pdfPadL - pdfPadR
)

// helvWidths holds Helvetica glyph widths in 1/1000 for ASCII 32..126.
var helvWidths = [128]float64{
	278, 278, 355, 556, 556, 889, 667, 191, 333, 333, 389, 584, 278, 333, 278, 278,
	556, 556, 556, 556, 556, 556, 556, 556, 556, 556, 278, 278, 584, 584, 584, 556,
	1015, 667, 667, 722, 722, 667, 611, 778, 722, 278, 500, 667, 556, 833, 722, 778,
	667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611, 278, 278, 278, 469, 556,
	333, 556, 556, 500, 556, 556, 278, 556, 556, 222, 222, 500, 222, 833, 556, 556,
	556, 556, 333, 500, 278, 556, 500, 722, 500, 500, 500, 334, 260, 334, 584, 0,
}

func textWidth(s string, size float64) float64 {
	total := 0.0
	for _, r := range s {
		if r >= 32 && r < 128 {
			total += helvWidths[r]
		} else {
			total += 556
		}
	}
	return total / 1000 * size
}

func wrapText(s string, size, maxWidth float64) []string {
	var lines []string
	for _, paragraph := range strings.Split(strings.ReplaceAll(strings.TrimSpace(s), "\r", ""), "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		line := ""
		for _, word := range words {
			candidate := word
			if line != "" {
				candidate = line + " " + word
			}
			if textWidth(candidate, size) <= maxWidth {
				line = candidate
				continue
			}
			if line != "" {
				lines = append(lines, line)
				line = word
			} else {
				lines = append(lines, word)
			}
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func winAnsiEncode(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 0x20 && r <= 0x7e:
			b.WriteByte(byte(r))
		case r == '\t':
			b.WriteByte(' ')
		case r >= 0xa0 && r <= 0xff:
			b.WriteByte(byte(r))
		case r == 0x20ac:
			b.WriteByte(0x80)
		case r == 0x201a:
			b.WriteByte(0x82)
		case r == 0x201e:
			b.WriteByte(0x84)
		case r == 0x2026:
			b.WriteByte(0x85)
		case r == 0x2018:
			b.WriteByte(0x91)
		case r == 0x2019:
			b.WriteByte(0x92)
		case r == 0x201c:
			b.WriteByte(0x93)
		case r == 0x201d:
			b.WriteByte(0x94)
		case r == 0x2022:
			b.WriteByte(0x95)
		case r == 0x2013:
			b.WriteByte(0x96)
		case r == 0x2014:
			b.WriteByte(0x97)
		default:
			b.WriteByte('?')
		}
	}
	return b.String()
}

func pdfText(s string) string {
	s = winAnsiEncode(s)
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\', '(', ')':
			b.WriteByte('\\')
			b.WriteByte(c)
		case 0x0a, 0x0d:
			b.WriteString("\\n")
		default:
			b.WriteByte(c)
		}
	}
	return "(" + b.String() + ")"
}

type pdfCanvas struct {
	ops []string
}

func (c *pdfCanvas) text(x, y float64, s string, size float64, bold bool, r, g, bl float64) {
	font := 1
	if bold {
		font = 2
	}
	c.ops = append(c.ops,
		fmt.Sprintf("q %.6g %.6g %.6g rg BT /F%d %.6g Tf 1 0 0 1 %.6g %.6g Tm %s Tj ET Q",
			r, g, bl, font, size, x, y, pdfText(s)))
}

func (c *pdfCanvas) rect(x, y, w, h float64, r, g, bl float64) {
	c.ops = append(c.ops, fmt.Sprintf("q %.6g %.6g %.6g rg %.6g %.6g %.6g %.6g re f Q", r, g, bl, x, y, w, h))
}

func (c *pdfCanvas) line(x1, y1, x2, y2 float64, width float64, r, g, bl float64) {
	c.ops = append(c.ops, fmt.Sprintf("q %.6g %.6g %.6g RG %.6g w %.6g %.6g m %.6g %.6g l S Q", r, g, bl, width, x1, y1, x2, y2))
}

func moneyIDR(value float64) string {
	amount := int(value + 0.5)
	digits := fmt.Sprintf("%d", amount)
	var b strings.Builder
	for i, ch := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(ch)
	}
	return "Rp " + b.String()
}

func formatIDRDate(t time.Time) string {
	return t.Format("02 Jan 2006")
}

// RenderInvoicePDF builds an A4 invoice as a PDF without external dependencies.
func RenderInvoicePDF(inv domain.Invoice) []byte {
	var canvas pdfCanvas
	top := pdfPageH - 48.0
	left := pdfPadL

	// Header: platform name + tagline
	name := inv.PlatformName
	if name == "" {
		name = "TKA"
	}
	canvas.text(left, top-20, name, 22, true, 0.047, 0.11, 0.25)
	tagline := inv.PlatformTagline
	if tagline != "" {
		for i, line := range wrapText(tagline, 10, pdfW) {
			if i >= 2 {
				break
			}
			canvas.text(left, top-20-26-float64(i)*13, line, 10, false, 0.42, 0.46, 0.54)
		}
	}

	// Right: INVOICE label
	rightTitle := "INVOICE"
	canvas.text(pdfPageW-pdfPadR-textWidth(rightTitle, 16), top-16, rightTitle, 16, true, 0.047, 0.11, 0.25)
	canvas.text(pdfPageW-pdfPadR-textWidth(inv.InvoiceNumber, 11), top-16-18, inv.InvoiceNumber, 11, false, 0.35, 0.37, 0.47)
	canvas.text(pdfPageW-pdfPadR-textWidth("Dibuat: "+formatIDRDate(inv.CreatedAt), 9), top-16-18-15, "Dibuat: "+formatIDRDate(inv.CreatedAt), 9, false, 0.55, 0.57, 0.64)

	// Divider
	divY := top - 78
	canvas.line(left, divY, pdfPageW-pdfPadR, divY, 1, 0.87, 0.9, 0.95)

	// Bill to + package
	y := divY - 34
	canvas.text(left, y, "Ditagihkan kepada", 9, true, 0.5, 0.53, 0.6)
	canvas.text(left, y-16, inv.BuyerName, 12, true, 0.15, 0.17, 0.24)
	canvas.text(left, y-16-16, inv.BuyerEmail, 10, false, 0.35, 0.37, 0.47)
	if inv.BuyerSchool != "" {
		canvas.text(left, y-16-16-14, "Jenjang "+inv.BuyerSchool, 10, false, 0.35, 0.37, 0.47)
	}

	midX := left + pdfW/2
	canvas.text(midX, y, "Paket", 9, true, 0.5, 0.53, 0.6)
	canvas.text(midX, y-16, inv.PackageTitle, 12, true, 0.15, 0.17, 0.24)
	kode := inv.PackageKode
	if kode != "" {
		canvas.text(midX, y-16-16, "Kode: "+kode, 10, false, 0.35, 0.37, 0.47)
	}
	meta := inv.PackageJenjang
	if inv.PackageValidity > 0 {
		if meta != "" {
			meta += " · "
		}
		meta += fmt.Sprintf("Aktif %d hari", inv.PackageValidity)
	}
	if meta != "" {
		canvas.text(midX, y-16-16-14, meta, 10, false, 0.35, 0.37, 0.47)
	}

	// Payment box
	boxY := y - 16 - 16 - 14 - 40
	canvas.rect(left, boxY-82, pdfW, 82, 0.956, 0.968, 1.0)

	// Tiga kolom: metode (kiri), status (tengah), tanggal bayar (kanan, rata kanan)
	colMethod := left + 18
	colStatus := left + pdfW*0.43
	colDateRight := pdfPageW - pdfPadR

	canvas.text(colMethod, boxY-16, "Metode pembayaran", 9, true, 0.5, 0.53, 0.6)
	canvas.text(colMethod, boxY-32, strings.ToUpper(inv.PaymentMethod), 10, true, 0.2, 0.24, 0.34)

	canvas.text(colStatus, boxY-16, "Status pembayaran", 9, true, 0.5, 0.53, 0.6)
	statusLabel := strings.ToUpper(inv.PaymentStatus)
	if inv.PaymentStatus == "paid" {
		statusLabel = "LUNAS"
	}
	canvas.text(colStatus, boxY-32, statusLabel, 10, true, 0.05, 0.55, 0.4)

	paidLabel := "Dibayar pada"
	var dateText string
	if inv.PaidAt != nil {
		dateText = formatIDRDate(*inv.PaidAt)
	} else {
		dateText = "—"
	}
	paidX := colDateRight - math.Max(textWidth(paidLabel, 9), textWidth(dateText, 10))
	canvas.text(paidX, boxY-16, paidLabel, 9, true, 0.5, 0.53, 0.6)
	canvas.text(paidX, boxY-32, dateText, 10, true, 0.2, 0.24, 0.34)

	// Total
	totalY := boxY - 82 - 120
	canvas.line(left, totalY+88, pdfPageW-pdfPadR, totalY+88, 1, 0.87, 0.9, 0.95)
	canvas.text(left, totalY+64, "Total", 12, true, 0.15, 0.17, 0.24)
	canvas.text(pdfPageW-pdfPadR-textWidth(moneyIDR(inv.Price), 18), totalY+64, moneyIDR(inv.Price), 18, true, 0.047, 0.11, 0.25)
	canvas.text(left, totalY+40, "Nomor invoice: "+inv.InvoiceNumber, 9, false, 0.5, 0.53, 0.6)
	canvas.text(left, totalY+26, "ID transaksi: "+inv.TransactionID, 9, false, 0.5, 0.53, 0.6)

	// Footer
	canvas.line(left, 70, pdfPageW-pdfPadR, 70, 1, 0.9, 0.92, 0.96)
	thankYou := "Terima kasih telah berbelanja di " + name + "."
	canvas.text(left, 52, thankYou, 9, false, 0.42, 0.46, 0.54)

	return buildPDF(canvas.ops)
}

func buildPDF(ops []string) []byte {
	content := strings.Join(ops, "\n") + "\n"
	// Objek PDF diberi nomor tetap; referensi memakai sintaks "N 0 R".
	fontF1 := "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>"
	fontF2 := "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>"
	contentBody := "<< /Length " + fmt.Sprintf("%d", len(content)) + " >>\nstream\n" + content + "endstream"
	pageBody := "<< /Type /Page /Parent 5 0 R /MediaBox [0 0 " + printNum(pdfPageW) + " " + printNum(pdfPageH) + "] /Resources << /Font << /F1 1 0 R /F2 2 0 R >> >> /Contents 3 0 R >>"
	pagesBody := "<< /Type /Pages /Kids [4 0 R] /Count 1 >>"
	catalogBody := "<< /Type /Catalog /Pages 5 0 R >>"

	bodies := []string{fontF1, fontF2, contentBody, pageBody, pagesBody, catalogBody}

	var buf strings.Builder
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(bodies))
	for i, body := range bodies {
		offsets[i] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n", i+1)
		buf.WriteString(body + "\n")
		buf.WriteString("endobj\n")
	}
	xrefPos := buf.Len()
	buf.WriteString("xref\n0 " + fmt.Sprintf("%d", len(bodies)+1) + "\n")
	buf.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offset)
	}
	buf.WriteString("trailer\n<< /Size " + fmt.Sprintf("%d", len(bodies)+1) + " /Root 6 0 R >>\nstartxref\n" + fmt.Sprintf("%d", xrefPos) + "\n%%EOF")
	return []byte(buf.String())
}

func printNum(v float64) string {
	return fmt.Sprintf("%.4g", v)
}
