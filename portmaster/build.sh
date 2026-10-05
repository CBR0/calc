#!/bin/sh
# Assemble the PortMaster port for the R36S (aarch64) into portmaster/dist:
#
#   dist/Calc.sh          the launch script
#   dist/calc/            the payload: binary, GTK3 libraries, fonts, ...
#
# Copy both to the ports folder of the device's GAMES partition
# (/roms/ports on AmberELEC).
set -e
cd "$(dirname "$0")/.."

# 1. The calculator itself (linux/arm64; CALC_FULLSCREEN=1 is read at run
#    time by the launch script).
go tool mygo build -platform linux/arm64

# 2. Assemble the port: clean first, then fetch the GTK3 runtime libraries
#    into dist/calc/libs.aarch64 (Ubuntu focal arm64; see fetch-libs.py).
DIST=portmaster/dist
rm -rf "$DIST"
mkdir -p "$DIST/calc/fonts" "$DIST/calc/licenses"
python3 portmaster/fetch-libs.py

cp build/linux-arm64/calc "$DIST/calc/calc.aarch64"
cp portmaster/port/Calc.sh "$DIST/Calc.sh"
cp portmaster/port/calc.gptk \
   portmaster/port/port.json \
   portmaster/port/README.md \
   portmaster/port/run.sh "$DIST/calc/"

cp /usr/share/fonts/TTF/DejaVuSans.ttf \
   /usr/share/fonts/TTF/DejaVuSans-Bold.ttf "$DIST/calc/fonts/"
cp /usr/share/licenses/ttf-dejavu/LICENSE "$DIST/calc/licenses/LICENSE.dejavu.txt"
cp LICENSE "$DIST/calc/licenses/LICENSE.calc.txt"
cp portmaster/port/licenses/LICENSE.libraries.txt "$DIST/calc/licenses/"

chmod +x "$DIST/Calc.sh" "$DIST/calc/run.sh"

# 640x480 (4:3) screenshot for the PortMaster catalogue, letterboxed.
if command -v magick > /dev/null 2>&1; then
  magick docs/screenshot.png -resize 480x480 -background "#1e1e1e" \
    -gravity center -extent 640x480 "$DIST/calc/screenshot.png"
fi

echo "dist ready:"
du -sh "$DIST"
find "$DIST" -maxdepth 2 | sort | head -20
