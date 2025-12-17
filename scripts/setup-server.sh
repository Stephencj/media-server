#!/bin/bash
#
# One-time server setup script for Pop!_OS / Ubuntu
# Run this on your Linux server to set up the media server
#

set -e

echo "=== Media Server Setup ==="
echo ""

# Check if running as root
if [ "$EUID" -eq 0 ]; then
    echo "Please run without sudo. Script will ask for sudo when needed."
    exit 1
fi

PROJECT_DIR="/opt/media-server"
WEBHOOK_PORT=9000

# Install Docker if not present
if ! command -v docker &> /dev/null; then
    echo "Installing Docker..."
    sudo apt-get update
    sudo apt-get install -y ca-certificates curl gnupg
    sudo install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    sudo chmod a+r /etc/apt/keyrings/docker.gpg

    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

    sudo apt-get update
    sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

    # Add current user to docker group
    sudo usermod -aG docker $USER
    echo "Docker installed. You may need to log out and back in for group changes."
else
    echo "Docker already installed"
fi

# Create project directory
echo "Setting up project directory..."
sudo mkdir -p "$PROJECT_DIR"
sudo chown $USER:$USER "$PROJECT_DIR"

# Clone or copy files
echo ""
echo "Project directory created at: $PROJECT_DIR"
echo ""
echo "Next steps:"
echo "1. Clone your repository:"
echo "   git clone https://github.com/stephencjuliano/media-server.git $PROJECT_DIR"
echo ""
echo "2. Create .env file:"
echo "   cp $PROJECT_DIR/.env.example $PROJECT_DIR/.env"
echo "   nano $PROJECT_DIR/.env"
echo ""

# Generate secrets
echo "Generating secrets..."
JWT_SECRET=$(openssl rand -hex 32)
WEBHOOK_SECRET=$(openssl rand -hex 16)

echo "3. Add these values to your .env file:"
echo "   JWT_SECRET=$JWT_SECRET"
echo "   WEBHOOK_SECRET=$WEBHOOK_SECRET"
echo ""

# Create symlink for media
echo "4. Create symlink to your media drive:"
echo "   sudo ln -s /media/$USER/YOUR_DRIVE_NAME /mnt/media"
echo ""

# Setup webhook service
echo "5. Setup webhook service (optional, for auto-deploy):"
cat << 'EOF'

Create /etc/systemd/system/media-server-webhook.service:

[Unit]
Description=Media Server Deployment Webhook
After=network.target docker.service

[Service]
Type=simple
User=$USER
Environment=WEBHOOK_PORT=9000
Environment=WEBHOOK_SECRET=YOUR_WEBHOOK_SECRET
Environment=COMPOSE_DIR=/opt/media-server
ExecStart=/usr/bin/python3 /opt/media-server/scripts/webhook-server.py
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target

Then run:
sudo systemctl daemon-reload
sudo systemctl enable media-server-webhook
sudo systemctl start media-server-webhook

EOF

echo ""
echo "6. Add GitHub Secrets:"
echo "   DEPLOY_WEBHOOK_URL=http://YOUR_SERVER_IP:$WEBHOOK_PORT/"
echo "   DEPLOY_WEBHOOK_SECRET=$WEBHOOK_SECRET"
echo ""
echo "=== Setup instructions complete ==="
