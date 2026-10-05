# PortMaster port (handhelds, aarch64)

Packages the calculator as a [PortMaster](https://portmaster.games/) port
for AmberELEC and friends on RK3326-class handhelds (R36S). The app runs
full screen under the **WestonPack** runtime (Wayland), with the GTK3
libraries it needs shipped in the port — the runtime provides cairo,
pango and the display stack, but no GTK.

## Build

```sh
sh portmaster/build.sh
```

This builds `linux/arm64`, downloads the GTK3 runtime libraries
(`fetch-libs.py`, Ubuntu focal arm64, pinned per soname), and assembles
`portmaster/dist/`:

```
dist/Calc.sh          launch script (goes to the ports folder)
dist/calc/            payload: calc.aarch64, libs.aarch64/, fonts/, ...
```

## Install on the device

Copy both to the `ports` folder of the GAMES partition (AmberELEC mounts
it at `/roms`):

```sh
cp portmaster/dist/Calc.sh  /run/media/$USER/GAMES/ports/
rsync -a portmaster/dist/calc/ /run/media/$USER/GAMES/ports/calc/
```

The port appears in the Ports menu. On first run it needs the
`weston_pkg_0.2` runtime in `PortMaster/libs` (PortMaster's runtime
manager, or the full installer, provides it).

## How it runs

`Calc.sh` (PortMaster's launch-script conventions) mounts the Weston
runtime and runs:

```
westonwrap.sh drm gl kiosk system  ./calc.aarch64
```

with `WRAPPED_LIBRARY_PATH=libs.aarch64` (our GTK3 stack takes precedence;
X11/Wayland/xkbcommon/GPU come from the runtime), `GDK_BACKEND=wayland`,
`GDK_GL=disable` (software drawing), a bundled DejaVu font through
`FONTCONFIG_FILE`, `CALC_FULLSCREEN=1` and `CALC_DRAWN_ICONS=1` (the
minimal font set has no ⧉ glyph). `gptokeyb` maps the gamepad: D-pad/left
stick move the pointer, A clicks, B/X/Y are Esc/Backspace/Enter, and
Start+Select quits.

GTK draws its built-in icons as PNG, so the port ships the gdk-pixbuf PNG
loader and `gdk-pixbuf-query-loaders`; `Calc.sh` writes `loaders.cache`
(with the port's own paths) on every run and passes it through
`GDK_PIXBUF_MODULE_FILE`. `GTK_CSD=0` keeps the window decorationless
under the kiosk shell.

The log of each run lands in `ports/calc/log.txt` on the device.

## Test on the device

```sh
systemctl stop emustation     # AmberELEC: free the display
/roms/ports/Calc.sh           # from SSH
systemctl start emustation
```
