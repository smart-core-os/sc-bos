#!/bin/sh
set -eu

# Substitute secrets into the UI config from environment variables before starting.
# Only the listed variables are substituted to avoid accidentally replacing other ${...} patterns.
envsubst '${OPENWEATHERMAP_API_KEY}' < /cfg/ui-config.json > /tmp/ui-config.json
mv /tmp/ui-config.json /cfg/ui-config.json

exec "$@"
