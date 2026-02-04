
FROM golang:1.25-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o jobloop ./cmd/api/

RUN go run github.com/playwright-community/playwright-go/cmd/playwright install --with-deps chromium


FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    xvfb \
    libgbm1 \
    libnss3 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libcups2 \
    libdrm2 \
    libxkbcommon0 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    libpango-1.0-0 \
    libcairo2 \
    libasound2 \
    libxshmfence1 \
    libx11-6 \
    libxcb1 \
    libxext6 \
    libglib2.0-0 \
    libdbus-1-3 \
    fonts-liberation \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean \
    && rm -rf /var/cache/apt/archives/*

WORKDIR /app

COPY --from=builder /app/jobloop .

COPY --from=builder /root/.cache/ms-playwright /root/.cache/ms-playwright
COPY --from=builder /root/.cache/ms-playwright-go /root/.cache/ms-playwright-go

RUN mkdir -p /app/logs

ENV ANTHROPIC_API_KEY="" \
    GOOGLE_API_KEY="" \
    DB_HOST="" \
    DB_PORT="5432" \
    DB_USER="" \
    DB_PASSWORD="" \
    DB_NAME="" \
    DISPLAY=:99

COPY --chmod=755 <<'EOF' /entrypoint.sh
#!/bin/bash
set -e

echo "=== JobLoop Scraper ==="
[ -n "$ANTHROPIC_API_KEY" ] && echo "✓ ANTHROPIC_API_KEY" || echo "⚠ ANTHROPIC_API_KEY missing"

# Start Xvfb
Xvfb $DISPLAY -screen 0 1920x1080x24 -ac &
sleep 2 && echo "✓ Xvfb started"

trap 'pkill -f Xvfb; exit 0' SIGTERM SIGINT
exec ./jobloop "$@"
EOF

ENTRYPOINT ["/entrypoint.sh"]