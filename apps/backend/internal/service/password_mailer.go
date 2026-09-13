package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

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
	serverAddr := m.host + ":" + m.port

	// Use TLS for SMTP connections to protect credentials in transit.
	// Falls back to STARTTLS if the server supports it.
	tlsConfig := &tls.Config{
		ServerName: m.host,
		MinVersion: tls.VersionTLS12,
	}

	var conn net.Conn
	var err error

	// Try direct TLS connection first (SMTPS on port 465)
	conn, err = tls.Dial("tcp", serverAddr, tlsConfig)
	if err != nil {
		// Fallback to plain TCP with STARTTLS upgrade
		conn, err = net.Dial("tcp", serverAddr)
		if err != nil {
			return fmt.Errorf("connect to SMTP server: %w", err)
		}
	}

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer func() { _ = client.Close() }()

	// If we used plain TCP, try to upgrade to STARTTLS
	if _, isTLS := conn.(*tls.Conn); !isTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("start TLS: %w", err)
			}
		}
	}

	// Authenticate if credentials are provided
	if m.username != "" {
		auth := smtp.PlainAuth("", m.username, m.password, m.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication: %w", err)
		}
	}

	// Set sender and recipient
	if err := client.Mail(m.from); err != nil {
		return fmt.Errorf("set sender: %w", err)
	}
	if err := client.Rcpt(recipient); err != nil {
		return fmt.Errorf("set recipient: %w", err)
	}

	// Write email body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("open data connection: %w", err)
	}
	body := "From: " + m.from + "\r\n" +
		"To: " + recipient + "\r\n" +
		"Subject: Reset kata sandi platform Tryout TKA\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" +
		"Kami menerima permintaan reset kata sandi akun Anda.\r\n\r\n" +
		"Buka tautan berikut dalam 15 menit:\r\n" + resetURL + "\r\n\r\n" +
		"Jika Anda tidak meminta reset, abaikan email ini."
	if _, err := w.Write([]byte(strings.ReplaceAll(body, "\n", "\r\n"))); err != nil {
		_ = w.Close()
		return fmt.Errorf("write email body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close data connection: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("quit SMTP: %w", err)
	}
	return nil
}
