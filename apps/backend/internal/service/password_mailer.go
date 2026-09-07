package service

import (
	"context"
	"fmt"
	"net/smtp"
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
	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}
	body := "From: " + m.from + "\r\n" +
		"To: " + recipient + "\r\n" +
		"Subject: Reset kata sandi TKA Juara\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" +
		"Kami menerima permintaan reset kata sandi akun TKA Juara Anda.\r\n\r\n" +
		"Buka tautan berikut dalam 15 menit:\r\n" + resetURL + "\r\n\r\n" +
		"Jika Anda tidak meminta reset, abaikan email ini."
	if err := smtp.SendMail(m.host+":"+m.port, auth, m.from, []string{recipient}, []byte(body)); err != nil {
		return fmt.Errorf("send password reset email: %w", err)
	}
	return nil
}
