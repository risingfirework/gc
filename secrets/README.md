# Production secrets

Create the following UTF-8 files locally. They are ignored by Git:

- `postgres_password.txt`: strong PostgreSQL password.
- `database_url.txt`: `postgres://tka:PASSWORD@postgres:5432/tka?sslmode=disable`.
- `redis_password.txt`: strong Redis password.
- `jwt_secret.txt`: at least 32 random characters.
- `payment_webhook_secret.txt`: the real payment-provider webhook secret.
- `sentry_dsn.txt`: Sentry project DSN for the Go API (an empty file disables Sentry).
- `grafana_admin_password.txt`: strong Grafana administrator password.
- `slack_webhook_url.txt`: Slack Incoming Webhook URL used by Alertmanager.
- `telegram_bot_token.txt`: Telegram Bot API token used by Alertmanager.
- `telegram_chat_id.txt`: numeric Telegram chat ID.
- `redis_exporter_passwords.json`: Redis exporter password map; see the operations playbook.

Use your platform's secret manager in real production. Keep file permissions restricted
to the deployment account and never commit secret values.
