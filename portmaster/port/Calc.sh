#!/bin/bash
# PORTMASTER: calc.zip, Calc.sh
#
# Calc (https://github.com/CBR0/calc) for PortMaster. The calculator is
# built for aarch64 and shows its GTK3 interface through the WestonPack
# runtime (Wayland), full screen. GTK3 and its own runtime libraries ship
# in calc/libs.aarch64: the runtime provides cairo, pango and the display
# stack, but no GTK.

XDG_DATA_HOME=${XDG_DATA_HOME:-$HOME/.local/share}

if [ -d "/opt/system/Tools/PortMaster/" ]; then
  controlfolder="/opt/system/Tools/PortMaster"
elif [ -d "/opt/tools/PortMaster/" ]; then
  controlfolder="/opt/tools/PortMaster"
elif [ -d "$XDG_DATA_HOME/PortMaster/" ]; then
  controlfolder="$XDG_DATA_HOME/PortMaster"
else
  controlfolder="/roms/ports/PortMaster"
fi

source $controlfolder/control.txt
source $controlfolder/device_info.txt

[ -f "${controlfolder}/mod_${CFW_NAME}.txt" ] && source "${controlfolder}/mod_${CFW_NAME}.txt"

get_controls

GAMEDIR=/$directory/ports/calc
CONFDIR="$GAMEDIR/conf/"

mkdir -p "$CONFDIR"

cd $GAMEDIR
> "$GAMEDIR/log.txt" && exec > >(tee "$GAMEDIR/log.txt") 2>&1

# Weston (Wayland) runtime
weston_dir=/tmp/weston
weston_runtime="weston_pkg_0.2"
if [ ! -f "$controlfolder/libs/${weston_runtime}.squashfs" ]; then
  if [ ! -f "$controlfolder/harbourmaster" ]; then
    pm_message "This port requires the latest PortMaster to run, please go to https://portmaster.games/ for more info."
    sleep 5
    exit 1
  fi
  $ESUDO $controlfolder/harbourmaster --quiet --no-check runtime_check "${weston_runtime}.squashfs"
fi
$ESUDO mkdir -p "${weston_dir}"
if [[ "$PM_CAN_MOUNT" != "N" ]]; then
  $ESUDO umount "${weston_dir}" 2>/dev/null || true
fi
$ESUDO mount "$controlfolder/libs/${weston_runtime}.squashfs" "${weston_dir}"

# Fonts for Pango/GTK: the bundled DejaVu, a config pointing at it and a
# cache inside the port's conf directory.
mkdir -p "$CONFDIR/fonts-cache"
cat > "$CONFDIR/fonts.conf" << EOF
<?xml version="1.0"?>
<fontconfig>
  <dir>$GAMEDIR/fonts</dir>
  <cachedir>$CONFDIR/fonts-cache</cachedir>
  <match target="pattern">
    <edit name="family" mode="prepend" binding="weak"><string>DejaVu Sans</string></edit>
  </match>
</fontconfig>
EOF

# gdk-pixbuf: GTK draws its built-in icons as PNG; the loader and the
# tool that writes loaders.cache (with this port's paths) ship with the
# app, and the cache is generated here on every run.
GDK_PIXBUF="$GAMEDIR/libs.${DEVICE_ARCH}/gdk-pixbuf"
LD_LIBRARY_PATH="$GAMEDIR/libs.${DEVICE_ARCH}" "$GDK_PIXBUF/gdk-pixbuf-query-loaders" \
  "$GDK_PIXBUF/loaders/"*.so > "$CONFDIR/loaders.cache" 2>/dev/null || true

# Also leave the cache where gdk-pixbuf looks by default, when the
# system's root is writable.
for d in /usr/lib/aarch64-linux-gnu/gdk-pixbuf-2.0/2.10.0 /usr/lib/gdk-pixbuf-2.0/2.10.0; do
  if mkdir -p "$d" 2>/dev/null; then
    cp "$CONFDIR/loaders.cache" "$d/loaders.cache" 2>/dev/null || true
  fi
done

export XDG_DATA_HOME="$CONFDIR"
export XDG_CONFIG_HOME="$CONFDIR"
export SDL_GAMECONTROLLERCONFIG="$sdl_controllerconfig"

# Controls (calc.gptk): D-pad/left stick move the pointer, A or R1 clicks,
# B is Escape (C), X is Backspace, Y or Start is Enter (=), L1 slows the
# pointer. Start+Select quits.
$ESUDO chmod 666 /dev/uinput 2>/dev/null
$ESUDO chmod +x "./calc.${DEVICE_ARCH}" 2>/dev/null

$GPTOKEYB "calc.aarch64" -c "./calc.gptk" &

pm_platform_helper "$GAMEDIR/calc.aarch64"

# Full screen under Weston's kiosk shell, through run.sh (which logs a few
# diagnostics first). GTK draws in software: the bundled libraries come
# first in the app's library path, and the runtime supplies the rest
# (X11, Wayland, xkbcommon, GPU).
$ESUDO env WRAPPED_LIBRARY_PATH="$GAMEDIR/libs.${DEVICE_ARCH}" CALC_FULLSCREEN=1 CALC_DRAWN_ICONS=1 XCURSOR_SIZE=32 \
  $weston_dir/westonwrap.sh drm gl kiosk system \
  GDK_BACKEND=wayland GDK_GL=disable GTK_CSD=0 FONTCONFIG_FILE="$CONFDIR/fonts.conf" GDK_PIXBUF_MODULE_FILE="$CONFDIR/loaders.cache" \
  ./run.sh

if [[ "$PM_CAN_MOUNT" != "N" ]]; then
  $ESUDO umount "${weston_dir}" 2>/dev/null || true
fi

pm_finish
