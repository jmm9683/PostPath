#!/bin/sh

set -e

CERT_PATH="/etc/letsencrypt/live/yourdomain.com/fullchain.pem"
CHECKSUM_FILE="/tmp/cert_checksum"

# Initialize checksum file if not present
if [ ! -f "$CHECKSUM_FILE" ]; then
  sha256sum "$CERT_PATH" > "$CHECKSUM_FILE"
fi

while :; do
 echo "[INFO] Running certbot renew..."
 certbot renew --webroot -w /var/www/certbot --quiet

 echo "[INFO] Checking if certificate changed..."
 NEW_CHECKSUM=$(sha256sum "$CERT_PATH")
 OLD_CHECKSUM=$(cat "$CHECKSUM_FILE")

 if [ "$NEW_CHECKSUM" != "$OLD_CHECKSUM" ]; then
  echo "[INFO] Certificate updated. Reloading nginx..."
  docker exec nginx nginx -s reload
  echo "$NEW_CHECKSUM" > "$CHECKSUM_FILE"
 else
  echo "[INFO] No change in certificate. Skipping reload."
 fi

 sleep 12h
done
#
