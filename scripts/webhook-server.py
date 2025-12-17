#!/usr/bin/env python3
"""
Simple webhook server for triggering Docker deployments.
Listens for POST requests and runs docker-compose to update the service.
"""

import http.server
import subprocess
import os
import sys
import logging
from pathlib import Path

# Configuration
PORT = int(os.environ.get('WEBHOOK_PORT', 9000))
SECRET = os.environ.get('WEBHOOK_SECRET', '')
COMPOSE_DIR = os.environ.get('COMPOSE_DIR', '/opt/media-server')
COMPOSE_FILE = os.environ.get('COMPOSE_FILE', 'docker-compose.prod.yml')

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class WebhookHandler(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        # Verify secret if configured
        if SECRET:
            auth_header = self.headers.get('Authorization', '')
            expected = f'Bearer {SECRET}'
            if auth_header != expected:
                logger.warning(f'Unauthorized request from {self.client_address[0]}')
                self.send_response(401)
                self.end_headers()
                self.wfile.write(b'Unauthorized')
                return

        logger.info(f'Received deployment webhook from {self.client_address[0]}')

        try:
            # Run deployment
            result = self.deploy()

            if result.returncode == 0:
                self.send_response(200)
                self.end_headers()
                self.wfile.write(b'Deployment successful')
                logger.info('Deployment completed successfully')
            else:
                self.send_response(500)
                self.end_headers()
                self.wfile.write(f'Deployment failed: {result.stderr}'.encode())
                logger.error(f'Deployment failed: {result.stderr}')

        except Exception as e:
            self.send_response(500)
            self.end_headers()
            self.wfile.write(f'Error: {str(e)}'.encode())
            logger.exception('Deployment error')

    def deploy(self):
        """Pull latest image and restart container."""
        compose_path = Path(COMPOSE_DIR) / COMPOSE_FILE

        if not compose_path.exists():
            raise FileNotFoundError(f'Compose file not found: {compose_path}')

        # Pull latest image
        logger.info('Pulling latest image...')
        pull_result = subprocess.run(
            ['docker', 'compose', '-f', str(compose_path), 'pull'],
            capture_output=True,
            text=True,
            cwd=COMPOSE_DIR
        )

        if pull_result.returncode != 0:
            return pull_result

        # Restart with new image
        logger.info('Restarting container...')
        up_result = subprocess.run(
            ['docker', 'compose', '-f', str(compose_path), 'up', '-d', '--remove-orphans'],
            capture_output=True,
            text=True,
            cwd=COMPOSE_DIR
        )

        return up_result

    def log_message(self, format, *args):
        # Suppress default HTTP logging, use our logger instead
        pass


def main():
    if not SECRET:
        logger.warning('WEBHOOK_SECRET not set - webhook is unprotected!')

    server_address = ('0.0.0.0', PORT)
    httpd = http.server.HTTPServer(server_address, WebhookHandler)

    logger.info(f'Webhook server listening on port {PORT}')
    logger.info(f'Compose directory: {COMPOSE_DIR}')
    logger.info(f'Compose file: {COMPOSE_FILE}')

    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        logger.info('Shutting down webhook server')
        httpd.shutdown()


if __name__ == '__main__':
    main()
