proxy_http_version 1.1;
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
# Timpa, jangan tambahkan header dari klien: mencegah spoofing X-Forwarded-For
# yang bisa mengakali rate limit per-IP di backend.
proxy_set_header X-Forwarded-For $remote_addr;
proxy_set_header X-Forwarded-Proto $scheme;
proxy_set_header X-Request-ID $request_id;
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection $connection_upgrade;
proxy_connect_timeout 3s;
proxy_send_timeout 30s;
proxy_read_timeout 30s;
proxy_hide_header X-Powered-By;
