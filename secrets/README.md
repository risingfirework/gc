# Production secrets

Create the following UTF-8 files locally. They are ignored by Git:

- `postgres_password.txt`: strong PostgreSQL password.
- `database_url.txt`: untuk PostgreSQL bawaan Compose gunakan `postgres://tka:PASSWORD@postgres:5432/tka?sslmode=disable` karena trafik hanya berada pada network Docker internal. Untuk database managed/eksternal wajib gunakan `sslmode=verify-full` dan CA penyedia.
- `redis_password.txt`: strong Redis password.
- `jwt_secret.txt`: at least 32 random characters.
- `payment_webhook_secret.txt`: Xendit webhook callback token (isikan nilai `x-callback-token` webhook). Untuk adapter HMAC lama, ini adalah signing secret.
- `xendit_secret_key.txt`: Xendit secret API key (awali `xnd_...`; kosongkan bila pembayaran belum aktif).
- `sentry_dsn.txt`: Sentry project DSN for the Go API (an empty file disables Sentry).
- `smtp_password.txt`: SMTP password for the `SMTP_USERNAME` account (leave empty when SMTP is disabled).
- `grafana_admin_password.txt`: strong Grafana administrator password.
- `slack_webhook_url.txt`: Slack Incoming Webhook URL used by Alertmanager.
- `telegram_bot_token.txt`: Telegram Bot API token used by Alertmanager.
- `telegram_chat_id.txt`: numeric Telegram chat ID.
- `redis_exporter_passwords.json`: Redis exporter password map; see the operations playbook.

Use your platform's secret manager in real production. Keep file permissions restricted
to the deployment account and never commit secret values.
