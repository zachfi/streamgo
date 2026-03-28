#!/usr/bin/env bash
# Run Zigbee2MQTT in Docker with /dev/ttyUSB0 to verify the dongle works.
# See docs/ZIGBEE_NETWORK_VALIDATION.md § "Verify dongle with Zigbee2MQTT (Docker)".

set -e

DATA_DIR="${Z2M_DATA_DIR:-$(pwd)/zigbee2mqtt-data}"
DEVICE="${Z2M_DEVICE:-/dev/ttyUSB0}"
TZ="${TZ:-America/New_York}"

mkdir -p "$DATA_DIR"

echo "Zigbee2MQTT starting (foreground). Frontend: http://localhost:9999"
echo "Data dir: $DATA_DIR  Device: $DEVICE"
echo "Press Ctrl+C to stop (container will be removed)"
docker run --rm \
  --name zigbee2mqtt \
  --device="$DEVICE:/dev/ttyUSB0" \
  -p 9999:8080 \
  -v "$DATA_DIR:/app/data" \
  -v /run/udev:/run/udev:ro \
  -e TZ="$TZ" \
  ghcr.io/koenkk/zigbee2mqtt:latest
