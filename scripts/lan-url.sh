#!/usr/bin/env sh
# Prints the address(es) friends on your LAN type into their browser to join,
# and checks that the game answers on each of them.
#
#   ./scripts/lan-url.sh
#
# Run it on the machine that hosts the stack (`docker compose up -d`).
set -eu

cd "$(dirname "$0")/.."

# Host port that nginx is published on (e.g. "0.0.0.0:8080" -> 8080).
port=$(docker compose port nginx 80 2>/dev/null | head -n1 | sed 's/.*://')
if [ -z "$port" ]; then
  echo "The stack is not running. Start it with: docker compose up -d" >&2
  exit 1
fi

# IPv4 addresses of real network interfaces (skip loopback, Docker bridges, veths).
ips=$(ip -4 -o addr show scope global \
  | awk '$2 !~ /^(docker|br-|veth|virbr|lo)/ { split($4, a, "/"); print a[1] }')
if [ -z "$ips" ]; then
  echo "No LAN address found. Are you connected to a network?" >&2
  exit 1
fi

echo "Friends on your network can join at:"
for ip in $ips; do
  url="http://$ip:$port"
  if curl -fsS --max-time 3 "$url/api/v1/health" >/dev/null 2>&1; then
    echo "  $url"
  else
    echo "  $url   (not answering — check that the stack is up)"
  fi
done
echo
echo "If friends can't connect, check the host firewall, e.g. 'sudo ufw status'."
