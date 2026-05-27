#!/usr/bin/env bash
set -euo pipefail

if [[ "${EUID}" -ne 0 ]]; then
  echo "Run as root" >&2
  exit 1
fi

APP_USER="turnable-manager"
APP_DIR="/opt/turnable-manager-panel"
STATE_DIR="/var/lib/turnable-manager-panel"
CONF_DIR="/etc/turnable-manager-panel"
TURNABLE_DIR="/opt/turnable-src"
PANEL_BIN="/usr/local/bin/turnable-manager-panel"
TURNABLE_BIN="/usr/local/bin/turnable"
ADMIN_FILE="/root/turnable-manager-admin.txt"

apt-get update
apt-get install -y ca-certificates curl git golang-go build-essential

if ! id "$APP_USER" >/dev/null 2>&1; then
  useradd --system --home "$STATE_DIR" --shell /usr/sbin/nologin "$APP_USER"
fi

mkdir -p "$APP_DIR" "$STATE_DIR" "$CONF_DIR" /etc/turnable
cp -a . "$APP_DIR/"
cd "$APP_DIR"

go test ./...
go build -trimpath -ldflags="-s -w" -o "$PANEL_BIN" ./cmd/turnable-manager-panel
chmod 0755 "$PANEL_BIN"

if [[ ! -x "$TURNABLE_BIN" ]]; then
  rm -rf "$TURNABLE_DIR"
  git clone --depth=1 https://github.com/TheAirBlow/Turnable.git "$TURNABLE_DIR"
  cd "$TURNABLE_DIR"
  go build -trimpath -ldflags="-s -w" -o "$TURNABLE_BIN" ./cmd
  chmod 0755 "$TURNABLE_BIN"
  cd "$APP_DIR"
fi

SESSION_SECRET="$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | xxd -p -c 64)"
ADMIN_PASSWORD="$(openssl rand -base64 18 2>/dev/null || head -c 18 /dev/urandom | base64)"

if [[ ! -f "$CONF_DIR/config.json" ]]; then
  cat > "$CONF_DIR/config.json" <<JSON
{
  "listen_address": "127.0.0.1:8899",
  "state_file": "$STATE_DIR/state.json",
  "public_url": "http://127.0.0.1:8899",
  "cookie_secure": false,
  "allow_service_control": true,
  "session_secret": "$SESSION_SECRET"
}
JSON
fi

cat > /etc/systemd/system/turnable-manager-panel.service <<SERVICE
[Unit]
Description=Turnable Manager Panel
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$APP_USER
Group=$APP_USER
Environment=TURNABLE_MANAGER_ADMIN_PASSWORD=$ADMIN_PASSWORD
ExecStart=$PANEL_BIN -config $CONF_DIR/config.json
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ReadWritePaths=$STATE_DIR /etc/turnable

[Install]
WantedBy=multi-user.target
SERVICE

cat > /etc/systemd/system/turnable.service <<SERVICE
[Unit]
Description=Turnable server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/etc/turnable
ExecStart=$TURNABLE_BIN server -c /etc/turnable/config.json -s /etc/turnable/store.json
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
SERVICE

chown -R "$APP_USER:$APP_USER" "$STATE_DIR" /etc/turnable
chmod 0700 "$STATE_DIR" /etc/turnable
chmod 0640 "$CONF_DIR/config.json"

echo "URL: http://127.0.0.1:8899/admin" > "$ADMIN_FILE"
echo "Username: admin" >> "$ADMIN_FILE"
echo "Password: $ADMIN_PASSWORD" >> "$ADMIN_FILE"
chmod 0600 "$ADMIN_FILE"

systemctl daemon-reload
systemctl enable --now turnable-manager-panel

echo "Installed turnable-manager-panel"
echo "Admin credentials saved to: $ADMIN_FILE"
echo "Panel listens on 127.0.0.1:8899. Put nginx in front of it before exposing it."
echo "After saving Turnable settings in the panel, press Apply, then run: systemctl enable --now turnable"
