#!/usr/bin/env bash
set -euo pipefail

PI_HOST="${PI_HOST:-luke@100.66.177.126}"
PI_KEY=/Users/luke/.ssh/id_surveillance_pi
PI_DIR="/home/luke/surveillance-client"
SERVICE="surveillance-client"

echo "==> Building frontend..."
cd web && npm run build && cd ..

echo "==> Cross-compiling for Pi (arm64)..."
GOOS=linux GOARCH=arm64 go build -o surveillance-client .

echo "==> Syncing to $PI_HOST..."
ssh -i "$PI_KEY" "$PI_HOST" "mkdir -p $PI_DIR"
scp -i "$PI_KEY" surveillance-client "$PI_HOST:$PI_DIR/surveillance-client.new"

echo "==> Swapping binary and restarting..."
ssh -i "$PI_KEY" "$PI_HOST" "cd $PI_DIR && mv surveillance-client.new surveillance-client && chmod +x surveillance-client && sudo systemctl restart $SERVICE"

echo "==> Done. Status:"
ssh -i "$PI_KEY" "$PI_HOST" "sudo systemctl status $SERVICE --no-pager -l" || true
