package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

const smtpSendTimeout = 20 * time.Second

type SMTPPasswordResetMailer struct {
	host, port, username, password, from string
}

func NewSMTPPasswordResetMailer(host, port, username, password, from string) *SMTPPasswordResetMailer {
	return &SMTPPasswordResetMailer{host: host, port: port, username: username, password: password, from: from}
}

func (m *SMTPPasswordResetMailer) SendPasswordReset(ctx context.Context, recipient, resetURL string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	host := strings.TrimSpace(m.host)
	from := strings.TrimSpace(m.from)
	if host == "" || from == "" {
		return errors.New("konfigurasi SMTP belum lengkap")
	}
	port := strings.TrimSpace(m.port)
	if port == "" {
		port = "587"
	}

	tlsConfig := &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	serverAddr := net.JoinHostPort(host, port)

	var conn net.Conn
	var err error
	if port == "465" {
		// Port 465 memakai TLS langsung (SMTPS).
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: tlsConfig}).DialContext(ctx, "tcp", serverAddr)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", serverAddr)
	}
	if err != nil {
		return fmt.Errorf("hubungi server SMTP: %w", err)
	}
	defer func() { _ = conn.Close() }()

	deadline := time.Now().Add(smtpSendTimeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	_ = conn.SetDeadline(deadline)

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("buat sesi SMTP: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Port selain 465 wajib memakai STARTTLS agar kredensial tidak terkirim
	// dalam bentuk plaintext.
	if _, alreadyTLS := conn.(*tls.Conn); !alreadyTLS {
		supportsSTARTTLS, _ := client.Extension("STARTTLS")
		if !supportsSTARTTLS {
			return errors.New("server SMTP tidak mendukung STARTTLS")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("mulai TLS: %w", err)
		}
	}

	if m.username != "" {
		auth := smtp.PlainAuth("", m.username, m.password, host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("autentikasi SMTP: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("alamat pengirim ditolak: %w", err)
	}
	if err := client.Rcpt(recipient); err != nil {
		return fmt.Errorf("alamat penerima ditolak: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("buka koneksi data email: %w", err)
	}
	if _, err := writer.Write([]byte(passwordResetMessage(from, recipient, resetURL))); err != nil {
		_ = writer.Close()
		return fmt.Errorf("tulis isi email: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("tutup koneksi data email: %w", err)
	}
	// Email sudah diterima server pada tahap ini. Kegagalan QUIT tidak perlu
	// dianggap gagal kirim karena bisa memicu pengiriman ulang.
	_ = client.Quit()
	return nil
}

func passwordResetMessage(from, recipient, resetURL string) string {
	lines := []string{
		"From: " + from,
		"To: " + recipient,
		"Subject: Reset kata sandi platform Tryout TKA",
		"Date: " + time.Now().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		"Halo,",
		"",
		"Kami menerima permintaan untuk mengatur ulang kata sandi akun Anda.",
		"Buka tautan berikut dalam 15 menit:",
		"",
		resetURL,
		"",
		"Jika Anda tidak meminta reset kata sandi, abaikan email ini.",
		"Kata sandi Anda tidak akan berubah sampai tautan tersebut dibuka.",
		"",
		"Salam,",
		"Tim SIAP TKA",
	}
	return strings.Join(lines, "\r\n") + "\r\n"
}
