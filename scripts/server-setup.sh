#!/bin/bash
# server-setup.sh - One-time setup for wtw on Linode
# Run as root: sudo bash scripts/server-setup.sh
set -e

DEPLOY_USER="deploy"
APP_DIR="/var/www/wtw"
SERVICE="wtw"

echo "=== What to Watch - Server Setup ==="
echo ""

# 1. Create deploy user (may already exist from moon)
if id "$DEPLOY_USER" &>/dev/null; then
    echo "[ok] User '$DEPLOY_USER' already exists"
else
    useradd -m -s /bin/bash "$DEPLOY_USER"
    echo "[ok] Created user '$DEPLOY_USER'"
fi

# 2. SSH key (may already exist from moon)
KEY_DIR="/home/$DEPLOY_USER/.ssh"
KEY_FILE="$KEY_DIR/github_actions_wtw"

mkdir -p "$KEY_DIR"
chmod 700 "$KEY_DIR"

if [ ! -f "$KEY_FILE" ]; then
    ssh-keygen -t ed25519 -f "$KEY_FILE" -N "" -C "github-actions-wtw-deploy"
    echo "[ok] Generated SSH key pair at $KEY_FILE"
else
    echo "[ok] SSH key already exists at $KEY_FILE"
fi

if ! grep -qF "$(cat "$KEY_FILE.pub")" "$KEY_DIR/authorized_keys" 2>/dev/null; then
    cat "$KEY_FILE.pub" >> "$KEY_DIR/authorized_keys"
    echo "[ok] Public key added to authorized_keys"
fi

chmod 600 "$KEY_DIR/authorized_keys"
chown -R "$DEPLOY_USER:$DEPLOY_USER" "$KEY_DIR"

# 3. Application directory
if [ -d "$APP_DIR" ]; then
    echo "[ok] Application directory $APP_DIR already exists"
else
    mkdir -p "$APP_DIR"
    chown www-data:www-data "$APP_DIR"
    echo "[ok] Created application directory $APP_DIR"
fi

# 4. Environment file
ENV_FILE="$APP_DIR/.env"
if [ -f "$ENV_FILE" ]; then
    echo "[ok] .env file already exists at $ENV_FILE (not overwriting)"
else
    cat > "$ENV_FILE" << 'ENV_TEMPLATE'
# Google OAuth
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GOOGLE_REDIRECT_URL=https://wtw.mchugh.au/auth/google/callback

# Session
SESSION_KEY=

# Claude API for recommendations
ANTHROPIC_API_KEY=

# TMDB for poster metadata
TMDB_API_KEY=

# Production
PROD=True
PORT=8991

# Monitor portal (optional)
MONITOR_URL=
MONITOR_API_KEY=
ENV_TEMPLATE
    chown www-data:www-data "$ENV_FILE"
    chmod 600 "$ENV_FILE"
    echo "[ok] Created .env template at $ENV_FILE (edit with real values)"
fi

# 5. Systemd service
SERVICE_FILE="/etc/systemd/system/$SERVICE.service"
cat > "$SERVICE_FILE" << 'SERVICE'
[Unit]
Description=What to Watch
After=network.target

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/var/www/wtw
EnvironmentFile=/var/www/wtw/.env
ExecStart=/var/www/wtw/wtw
Restart=on-failure
RestartSec=5

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true

[Install]
WantedBy=multi-user.target
SERVICE

systemctl daemon-reload
echo "[ok] Created systemd service at $SERVICE_FILE"

# 6. Deploy script
DEPLOY_SCRIPT="/usr/local/bin/deploy-wtw"
if [ -f "scripts/deploy-wtw" ]; then
    install -m 755 "scripts/deploy-wtw" "$DEPLOY_SCRIPT"
    echo "[ok] Installed $DEPLOY_SCRIPT from local"
else
    DEPLOY_SCRIPT_URL="https://raw.githubusercontent.com/exploded/wtw/main/scripts/deploy-wtw"
    if curl -fsSL "$DEPLOY_SCRIPT_URL" -o "$DEPLOY_SCRIPT"; then
        chmod +x "$DEPLOY_SCRIPT"
        echo "[ok] Installed $DEPLOY_SCRIPT from GitHub"
    else
        echo "[warn] Could not download deploy-wtw, install manually"
    fi
fi

# 7. Sudoers
SUDOERS_FILE="/etc/sudoers.d/wtw-deploy"
cat > "$SUDOERS_FILE" << 'EOF'
deploy ALL=(ALL) NOPASSWD: /usr/local/bin/deploy-wtw
deploy ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop wtw
EOF

chmod 440 "$SUDOERS_FILE"
visudo -c -f "$SUDOERS_FILE"
echo "[ok] sudoers entry created at $SUDOERS_FILE"

# 8. Caddy config reminder
echo ""
echo "=== Setup complete ==="
echo ""
echo "Add to your Caddyfile:"
echo ""
echo "  wtw.mchugh.au {"
echo "      reverse_proxy localhost:8991"
echo "  }"
echo ""
echo "Then reload: sudo systemctl reload caddy"
echo ""
echo "Add these GitHub repository secrets:"
echo ""
echo "Secret: DEPLOY_HOST = $(hostname -I | awk '{print $1}')"
echo "Secret: DEPLOY_USER = $DEPLOY_USER"
echo "Secret: DEPLOY_SSH_KEY = (paste private key below)"
echo ""
echo "---BEGIN PRIVATE KEY---"
cat "$KEY_FILE"
echo "---END PRIVATE KEY---"
echo ""
echo "Edit $ENV_FILE with real API keys before first deploy."
