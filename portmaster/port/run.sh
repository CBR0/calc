#!/bin/sh
# Calc launcher for the handheld port: logs a few diagnostics, then runs
# the calculator. Used as the command of westonwrap.sh, so it inherits the
# app's environment (GDK_PIXBUF_MODULE_FILE, FONTCONFIG_FILE, ...).
cd "$(dirname "$0")" || exit 1

echo "--- calc diagnostics ---"
echo "GDK_PIXBUF_MODULE_FILE=${GDK_PIXBUF_MODULE_FILE:-unset}"
echo "FONTCONFIG_FILE=${FONTCONFIG_FILE:-unset}"
echo "LD_LIBRARY_PATH=$LD_LIBRARY_PATH"
ls -la "${GDK_PIXBUF_MODULE_FILE:-/nonexistent}" 2>&1

# Load a PNG through gdk-pixbuf with the same environment: if this fails,
# image loaders are not being registered for the app either.
if [ -x ./libs.aarch64/gdk-pixbuf/gdk-pixbuf-pixdata ] && [ -f ./screenshot.png ]; then
  rm -f /tmp/pixdata.out /tmp/pixdata.log
  LD_LIBRARY_PATH="$(pwd)/libs.aarch64" \
  GDK_PIXBUF_MODULE_FILE="$GDK_PIXBUF_MODULE_FILE" \
    ./libs.aarch64/gdk-pixbuf/gdk-pixbuf-pixdata ./screenshot.png /tmp/pixdata.out \
    > /tmp/pixdata.log 2>&1
  code=$?
  bytes=$(wc -c < /tmp/pixdata.out 2>/dev/null || echo 0)
  echo "pixdata exit=$code out=$bytes bytes"
  head -3 /tmp/pixdata.log 2>/dev/null
fi

exec ./calc.aarch64
