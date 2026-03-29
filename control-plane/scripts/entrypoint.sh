#!/bin/sh
set -eu

echo "startup: controlplane entrypoint begin"
echo "startup: launching /usr/local/bin/controlplane (DB wait + baseline SQL replay in app.Boot — not a versioned DB upgrade)"
exec /usr/local/bin/controlplane
