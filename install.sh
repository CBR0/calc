#!/bin/sh
# Calc — installs the latest release for the current user, no root needed.
#
#   curl -fsSL https://raw.githubusercontent.com/CBR0/calc/main/install.sh | sh
#
# It downloads the release for this machine (or builds it from the source
# when there is no release archive for it yet) and installs the app in
# ~/.local/calc.app, the `calc` command in ~/.local/bin and its entry in the
# applications menu. Running it again updates to the latest release.
#
#   sh install.sh                                  # latest release
#   sh install.sh calc-0.1.0-linux-amd64.tar.gz    # install a local archive
#   sh install.sh --uninstall
set -eu

repo='CBR0/calc'
name='calc'
app_name='Calc'

app_dir="$HOME/.local/$name.app"
bin_dir="$HOME/.local/bin"
data_home="${XDG_DATA_HOME:-$HOME/.local/share}"
desktop_dir="$data_home/applications"
desktop_file="$desktop_dir/$name.desktop"

say() { printf '%s\n' "$*"; }
fail() {
	printf 'install.sh: %s\n' "$*" >&2
	exit 1
}

usage() {
	cat <<EOF
Usage: sh install.sh [ARCHIVE | --uninstall]

Installs $app_name for the current user in ~/.local. Without arguments it
downloads the latest release of github.com/$repo, or builds it from the
source when there is no release archive for this machine. Pass a .tar.gz
archive to install that one instead.
EOF
}

update_databases() {
	if command -v update-desktop-database >/dev/null 2>&1; then
		update-desktop-database "$desktop_dir" >/dev/null 2>&1 || true
	fi
	if command -v kbuildsycoca6 >/dev/null 2>&1; then
		kbuildsycoca6 --noincremental >/dev/null 2>&1 || true
	elif command -v kbuildsycoca5 >/dev/null 2>&1; then
		kbuildsycoca5 --noincremental >/dev/null 2>&1 || true
	fi
}

uninstall() {
	rm -rf "$app_dir" "$bin_dir/$name" "$desktop_file"
	update_databases
	say "Removed $app_name."
}

case "${1:-}" in
-h | --help)
	usage
	exit 0
	;;
--uninstall)
	[ "$(id -u)" != 0 ] || fail "run this as the user to uninstall $app_name for, not as root"
	uninstall
	exit 0
	;;
-*)
	usage >&2
	exit 2
	;;
esac

[ "$(uname -s)" = Linux ] || fail "$app_name installs on Linux"
[ "$(id -u)" != 0 ] || fail "run this as the user to install $app_name for, not as root"

case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) fail "$app_name is not made for $(uname -m)" ;;
esac

fetch() {
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$1"
	elif command -v wget >/dev/null 2>&1; then
		wget -qO- "$1"
	else
		fail "downloading needs curl or wget"
	fi
}

download() { # url file
	if command -v curl >/dev/null 2>&1; then
		curl -fL --progress-bar -o "$2" "$1"
	elif command -v wget >/dev/null 2>&1; then
		wget -O "$2" "$1"
	fi
}

# release_url reads the JSON of the GitHub releases API on stdin and prints
# the download URL of the first archive for linux-$arch, if any.
release_url() {
	grep -o '"browser_download_url": *"[^"]*"' |
		sed 's/.*"\(https[^"]*\)"$/\1/' |
		grep -E "/$name-[^/]*-linux-$arch\.tar\.gz\$" |
		head -n 1
}

# ours reports whether path is a link into the app's own directory.
ours() {
	[ -L "$1" ] || return 1
	case "$(readlink "$1")" in
	"$app_dir"/*) return 0 ;;
	*) return 1 ;;
	esac
}

work=$(mktemp -d)
staging="$HOME/.local/.$name.app.install"
trap 'rm -rf "$work" "$staging"' EXIT
trap 'exit 1' HUP INT TERM

archive="${1:-}"

if [ -z "$archive" ]; then
	api="https://api.github.com/repos/$repo/releases"
	say "Looking for the latest $app_name release of $repo"
	url=$(fetch "$api/latest" 2>/dev/null | release_url || true)
	if [ -z "$url" ]; then
		# No release marked as latest: take the newest one with an archive.
		url=$(fetch "$api?per_page=30" 2>/dev/null | release_url || true)
	fi
	if [ -n "$url" ]; then
		say "Downloading $url"
		download "$url" "$work/$name.tar.gz" || fail "could not download $url"
		archive="$work/$name.tar.gz"
	else
		# No release archive yet: build from the source, when Go is around.
		command -v git >/dev/null 2>&1 || fail "no release archive for linux-$arch: install git and Go, or download one yourself"
		command -v go >/dev/null 2>&1 || fail "no release archive for linux-$arch: install Go, or download one yourself"
		say "No release archive for linux-$arch: building from source"
		git clone --depth 1 "https://github.com/$repo.git" "$work/src" >/dev/null 2>&1 ||
			fail "could not clone https://github.com/$repo"
		(cd "$work/src" && go tool mygo build -platform "linux/$arch") ||
			fail "the build failed"
		for f in "$work/src/build/linux-$arch/$name-"*"-linux-$arch.tar.gz"; do
			archive="$f"
		done
	fi
fi

[ -f "$archive" ] || fail "no archive at $archive"
tar -tzf "$archive" >"$work/names" 2>/dev/null || fail "$archive is not a .tar.gz archive"
if grep -Eq '^/|(^|/)\.\.(/|$)' "$work/names"; then
	fail "$archive holds files outside the app"
fi

say "Installing $app_name in $app_dir"
rm -rf "$staging"
mkdir -p "$staging"
tar -xzf "$archive" -C "$staging"
[ -f "$staging/$name" ] && [ -x "$staging/$name" ] || fail "$archive is not $app_name: it holds no $name"

# Replace the app as a whole, so that no file of another version stays.
rm -rf "$staging.old"
if [ -e "$app_dir" ]; then
	mv "$app_dir" "$staging.old"
fi
mv "$staging" "$app_dir"
rm -rf "$staging.old"

# The command in ~/.local/bin, unless something else already owns the name.
bin_link="$bin_dir/$name"
if { [ -e "$bin_link" ] || [ -L "$bin_link" ]; } && ! ours "$bin_link"; then
	say "Leaving $bin_link alone: it is not $app_name's"
	run="$bin_link"
else
	mkdir -p "$bin_dir"
	ln -sf "$app_dir/$name" "$bin_link"
	run="$bin_link"
	if [ "$(command -v "$name" || true)" = "$bin_link" ]; then
		run="$name"
	fi
fi

# The menu entry, running the app and showing its icon by their paths.
if [ -f "$app_dir/$name.desktop" ]; then
	mkdir -p "$desktop_dir"
	while IFS= read -r line || [ -n "$line" ]; do
		case "$line" in
		"Exec=$name" | "Exec=$name "*) line="Exec=\"$app_dir/$name\"${line#"Exec=$name"}" ;;
		"Icon=$name") [ ! -f "$app_dir/$name.png" ] || line="Icon=$app_dir/$name.png" ;;
		esac
		printf '%s\n' "$line"
	done <"$app_dir/$name.desktop" >"$desktop_file"
fi
update_databases

say "Installed $app_name: open it from the applications menu, or run $run"
