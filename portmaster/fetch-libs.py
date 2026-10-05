#!/usr/bin/env python3
"""Fetch the aarch64 GTK3 runtime libraries the calculator needs.

AmberELEC has no GTK3 and the WestonPack runtime ships cairo/pango but not
GTK/GLib. This script downloads the runtime pieces of the GTK3 stack from
Ubuntu 20.04 (focal) for arm64 -- glibc 2.31 is old enough for AmberELEC --
resolves the dependency closure with readelf and places one file per needed
soname in dist/calc/libs.aarch64.

Libraries that WestonPack already provides through its own library path
(X11, Wayland, GPU/input core) are left out.
"""

import gzip
import os
import re
import shutil
import subprocess
import sys
import urllib.request

HERE = os.path.dirname(os.path.abspath(__file__))
DIST = os.path.join(HERE, "dist", "calc")
LIBS = os.path.join(DIST, "libs.aarch64")
CACHE = os.path.join(HERE, ".cache")
ARCHIVE = "http://ports.ubuntu.com/ubuntu-ports"
SUITES = ["focal", "focal-updates", "focal-security"]

# soname -> Ubuntu package (focal, main)
PKG = {
    "libgtk-3.so.0": "libgtk-3-0",
    "libgdk-3.so.0": "libgtk-3-0",
    "libglib-2.0.so.0": "libglib2.0-0",
    "libgobject-2.0.so.0": "libglib2.0-0",
    "libgio-2.0.so.0": "libglib2.0-0",
    "libgmodule-2.0.so.0": "libglib2.0-0",
    "libgthread-2.0.so.0": "libglib2.0-0",
    "libgdk_pixbuf-2.0.so.0": "libgdk-pixbuf2.0-0",
    "libatk-1.0.so.0": "libatk1.0-0",
    "libcairo.so.2": "libcairo2",
    "libcairo-gobject.so.2": "libcairo-gobject2",
    "libatk-bridge-2.0.so.0": "libatk-bridge2.0-0",
    "libatspi.so.0": "libatspi2.0-0",
    "libdbus-1.so.3": "libdbus-1-3",
    "libsystemd.so.0": "libsystemd0",
    "liblz4.so.1": "liblz4-1",
    "libgcrypt.so.20": "libgcrypt20",
    "libgpg-error.so.0": "libgpg-error0",
    "liblzma.so.5": "liblzma5",
    "libzstd.so.1": "libzstd1",
    "libcap.so.2": "libcap2",
    "libapparmor.so.1": "libapparmor1",
    "libpcre.so.3": "libpcre3",
    "libcloudproviders.so.0": "libcloudproviders0",
    "libpango-1.0.so.0": "libpango-1.0-0",
    "libpangocairo-1.0.so.0": "libpangocairo-1.0-0",
    "libpangoft2-1.0.so.0": "libpangoft2-1.0-0",
    "libharfbuzz.so.0": "libharfbuzz0b",
    "libfontconfig.so.1": "libfontconfig1",
    "libfreetype.so.6": "libfreetype6",
    "libexpat.so.1": "libexpat1",
    "libpng16.so.16": "libpng16-16",
    "libpixman-1.so.0": "libpixman-1-0",
    "libfribidi.so.0": "libfribidi0",
    "libthai.so.0": "libthai0",
    "libdatrie.so.1": "libdatrie1",
    "libgraphite2.so.3": "libgraphite2-3",
    "libpcre2-8.so.0": "libpcre2-8-0",
    "libz.so.1": "zlib1g",
    "libmount.so.1": "libmount1",
    "libblkid.so.1": "libblkid1",
    "libuuid.so.1": "libuuid1",
    "libselinux.so.1": "libselinux1",
    "libffi.so.7": "libffi7",
    "libbz2.so.1.0": "libbz2-1.0",
    "libbrotlidec.so.1": "libbrotli1",
    "libbrotlicommon.so.1": "libbrotli1",
    "libjpeg.so.8": "libjpeg-turbo8",
    "libtiff.so.5": "libtiff5",
    "libwebp.so.6": "libwebp6",
    "libgcc_s.so.1": "libgcc-s1",
    # GL/EGL: only to satisfy linkage; the app draws in software
    # (GDK_GL=disable) and never creates a GL context.
    "libEGL.so.1": "libegl1",
    "libGLdispatch.so.0": "libglvnd0",
    "libGLX.so.0": "libglx0",
}

# WestonPack provides these through its own library path (lib_aarch64 and
# extra_wayland are in the app's LD_LIBRARY_PATH after ours).
DEFER_PREFIXES = (
    "libX11", "libXau.so", "libXdmcp", "libXext", "libXrender", "libXi.so",
    "libXfixes", "libXcursor", "libXrandr", "libXinerama", "libXcomposite",
    "libXdamage", "libXss", "libXtst", "libXxf86", "libXfont", "libXft",
    "libxcb", "libwayland", "libxkbcommon", "libepoxy",
)
# glibc and kernel-provided names never ship in the bundle.
GLIBC = {
    "libc.so.6", "libm.so.6", "libdl.so.2", "libpthread.so.0", "librt.so.1",
    "libresolv.so.2", "libutil.so.1", "libanl.so.1", "ld-linux-aarch64.so.1",
    "linux-vdso.so.1",
}

SEED = [
    "libgtk-3.so.0", "libgdk-3.so.0", "libglib-2.0.so.0",
    "libgobject-2.0.so.0", "libgio-2.0.so.0", "libgmodule-2.0.so.0",
    "libgdk_pixbuf-2.0.so.0", "libatk-1.0.so.0", "libcairo.so.2",
    "libpango-1.0.so.0", "libpangocairo-1.0.so.0", "libpangoft2-1.0.so.0",
]


def run(*cmd, **kw):
    return subprocess.run(cmd, check=True, **kw)


def download(url, dest):
    if os.path.exists(dest) and os.path.getsize(dest) > 0:
        return dest
    os.makedirs(os.path.dirname(dest), exist_ok=True)
    print("  get", url)
    with urllib.request.urlopen(url) as r, open(dest + ".part", "wb") as f:
        shutil.copyfileobj(r, f)
    os.replace(dest + ".part", dest)
    return dest


def packages():
    """package -> pool filename, from the focal indexes (main and universe)."""
    out = {}
    for suite in SUITES:
        for comp in ("main", "universe"):
            url = f"{ARCHIVE}/dists/{suite}/{comp}/binary-arm64/Packages.gz"
            dest = os.path.join(CACHE, suite, comp, "Packages.gz")
            try:
                download(url, dest)
            except urllib.error.HTTPError:
                continue
            with gzip.open(dest, "rt", errors="replace") as f:
                cur = {}
                for line in f:
                    line = line.rstrip("\n")
                    if not line:
                        if cur.get("Package"):
                            out[cur["Package"]] = cur["Filename"]
                        cur = {}
                        continue
                    if line.startswith("Package: "):
                        cur["Package"] = line[9:]
                    elif line.startswith("Filename: "):
                        cur["Filename"] = line[10:]
                if cur.get("Package"):
                    out[cur["Package"]] = cur["Filename"]
    return out


def needed(path):
    out = subprocess.run(["readelf", "-d", path], capture_output=True, text=True).stdout
    return re.findall(r"\(NEEDED\)\s+Shared library: \[([^\]]+)\]", out)


def main():
    idx = packages()
    os.makedirs(LIBS, exist_ok=True)
    extracted = {}
    bundled = []

    def extract(pkg):
        "Extract a package's /usr/lib/aarch64-linux-gnu libraries into the cache."
        root = os.path.join(CACHE, "root", pkg)
        if pkg in extracted:
            return root
        if pkg not in idx:
            sys.exit(f"error: package {pkg} not found in the focal indexes")
        deb = os.path.join(CACHE, "deb", os.path.basename(idx[pkg]))
        download(ARCHIVE + "/" + idx[pkg], deb)
        tmp = os.path.join(CACHE, "x", pkg)
        os.makedirs(tmp, exist_ok=True)
        run("ar", "x", deb, cwd=tmp)
        data = [f for f in os.listdir(tmp) if f.startswith("data.tar")]
        run("tar", "xf", os.path.join(tmp, data[0]), "-C", tmp)
        shutil.rmtree(root, ignore_errors=True)
        # Ubuntu keeps libraries under /usr/lib/aarch64-linux-gnu, and some
        # (zlib, libgcc) under /lib/aarch64-linux-gnu.
        first = True
        got = False
        for sub in ("usr/lib/aarch64-linux-gnu", "lib/aarch64-linux-gnu"):
            src = os.path.join(tmp, sub)
            if not os.path.isdir(src):
                continue
            shutil.copytree(src, root, symlinks=True, dirs_exist_ok=not first,
                            ignore=shutil.ignore_patterns("*.a", "*.la"))
            first, got = False, True
        if not got:
            sys.exit(f"error: {pkg} has no aarch64-linux-gnu libraries")
        shutil.rmtree(tmp, ignore_errors=True)
        extracted[pkg] = root
        return root

    def bundle(soname, root):
        "Copy soname's real file into the libs dir and scan its needs."
        src = os.path.join(root, soname)
        if not os.path.exists(src):
            sys.exit(f"error: {soname} not in package root {root}")
        dst = os.path.join(LIBS, soname)
        shutil.copyfile(os.path.realpath(src), dst)
        bundled.append(dst)
        return needed(dst)

    queue = list(SEED)
    seen = set()
    while queue:
        soname = queue.pop(0)
        if soname in seen:
            continue
        seen.add(soname)
        if soname in GLIBC or soname.startswith(DEFER_PREFIXES):
            continue
        pkg = PKG.get(soname)
        if not pkg:
            sys.exit(f"error: no package mapped for {soname}; add it to PKG")
        print(f"* {soname} ({pkg})")
        root = extract(pkg)
        for dep in bundle(soname, root):
            if dep not in seen:
                queue.append(dep)

    # Report: every NEEDED of the bundle must be bundled, deferred or glibc.
    missing = set()
    for lib in bundled:
        for dep in needed(lib):
            if dep in seen:
                continue
            if os.path.exists(os.path.join(LIBS, dep)):
                continue
            if dep in GLIBC or dep.startswith(DEFER_PREFIXES):
                continue
            missing.add((os.path.basename(lib), dep))
    if missing:
        print("\nUNRESOLVED:")
        for lib, dep in sorted(missing):
            print(f"  {lib} needs {dep}")
        sys.exit(1)

    # gdk-pixbuf image loaders. GTK draws its built-in icons as PNG, so
    # without the PNG loader it aborts when an icon falls back to
    # "image-missing". The loader and gdk-pixbuf-query-loaders (which
    # writes loaders.cache with this port's paths on the device) go in
    # libs.aarch64/gdk-pixbuf.
    pix = os.path.join(CACHE, "root", "libgdk-pixbuf2.0-0",
                       "gdk-pixbuf-2.0", "2.10.0", "loaders", "libpixbufloader-png.so")
    if not os.path.exists(pix):
        sys.exit("error: libpixbufloader-png.so missing from libgdk-pixbuf2.0-0")
    loaders = os.path.join(LIBS, "gdk-pixbuf", "loaders")
    os.makedirs(loaders, exist_ok=True)
    shutil.copyfile(pix, os.path.join(loaders, "libpixbufloader-png.so"))

    # gdk-pixbuf-query-loaders writes loaders.cache; it ships inside
    # libgdk-pixbuf2.0-0 under gdk-pixbuf-2.0/.
    ql = os.path.join(CACHE, "root", "libgdk-pixbuf2.0-0",
                      "gdk-pixbuf-2.0", "gdk-pixbuf-query-loaders")
    if not os.path.exists(ql):
        sys.exit("error: gdk-pixbuf-query-loaders missing from libgdk-pixbuf2.0-0")
    dst = os.path.join(LIBS, "gdk-pixbuf", "gdk-pixbuf-query-loaders")
    shutil.copyfile(ql, dst)
    os.chmod(dst, 0o755)

    # gdk-pixbuf-pixdata loads an image through the same module machinery:
    # the port's launcher uses it to check the loaders on the device.
    pkg = "libgdk-pixbuf2.0-bin"
    if pkg not in idx:
        sys.exit(f"error: package {pkg} not found in the focal indexes")
    deb = os.path.join(CACHE, "deb", os.path.basename(idx[pkg]))
    download(ARCHIVE + "/" + idx[pkg], deb)
    tmp = os.path.join(CACHE, "x", pkg)
    os.makedirs(tmp, exist_ok=True)
    run("ar", "x", deb, cwd=tmp)
    data = [f for f in os.listdir(tmp) if f.startswith("data.tar")]
    run("tar", "xf", os.path.join(tmp, data[0]), "-C", tmp)
    src = os.path.join(tmp, "usr", "bin", "gdk-pixbuf-pixdata")
    if not os.path.exists(src):
        sys.exit("error: gdk-pixbuf-pixdata missing from " + pkg)
    dst = os.path.join(LIBS, "gdk-pixbuf", "gdk-pixbuf-pixdata")
    shutil.copyfile(src, dst)
    os.chmod(dst, 0o755)
    shutil.rmtree(tmp, ignore_errors=True)

    # The loader and the tools need the same runtime as the rest.
    for extra in (os.path.join(loaders, "libpixbufloader-png.so"),
                  os.path.join(LIBS, "gdk-pixbuf", "gdk-pixbuf-query-loaders"),
                  os.path.join(LIBS, "gdk-pixbuf", "gdk-pixbuf-pixdata")):
        for dep in needed(extra):
            if dep in GLIBC or os.path.exists(os.path.join(LIBS, dep)):
                continue
            sys.exit(f"error: {os.path.basename(extra)} needs {dep}, not bundled")

    print(f"\n{len(bundled)} libraries in {LIBS}")
    print("all NEEDED entries bundled, deferred to WestonPack, or glibc")
    print("gdk-pixbuf: PNG loader + gdk-pixbuf-query-loaders in libs.aarch64/gdk-pixbuf")


if __name__ == "__main__":
    main()
