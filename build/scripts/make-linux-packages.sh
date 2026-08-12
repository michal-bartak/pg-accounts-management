#!/usr/bin/env bash
# Build Linux .deb and .rpm packages from the binary in build/bin, using fpm.
#
#   OUTPUT=DbAccounts VERSION=0.3.0 ARCH_LABEL=amd64 build/scripts/make-linux-packages.sh
#
# Installs: /usr/bin/<OUTPUT>, a .desktop entry and an icon. Runtime dependencies name
# the WebKit build the binary is linked against (webkit2gtk 4.1 — see the Makefile's
# webkit2_41 tag). Run from the repo root; writes both packages under dist/.
set -euo pipefail

OUTPUT="${OUTPUT:-DbAccounts}"
VERSION="${VERSION:-$(tr -d ' \n\r' < VERSION)}"
ARCH_LABEL="${ARCH_LABEL:-amd64}"
DIST="${DIST:-dist}"

PKGNAME="dbaccounts"
DESCRIPTION="Maintain PostgreSQL roles across many clusters"
URL="https://github.com/michal-bartak/pg-accounts-management"
# Package metadata is public; override MAINTAINER to put a real address in the packages.
MAINTAINER="${MAINTAINER:-Michal Bartak <michal-bartak@users.noreply.github.com>}"

binary="build/bin/$OUTPUT"
[ -f "$binary" ] || { echo "ERROR: $binary not found — run 'make build' first"; exit 1; }
command -v fpm >/dev/null 2>&1 || {
  echo "ERROR: fpm not found. Install it with:"
  echo "  sudo apt-get install -y rpm ruby-dev build-essential && sudo gem install --no-document fpm"
  exit 1
}

mkdir -p "$DIST"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

cp build/linux/dbaccounts.desktop "$work/$PKGNAME.desktop"

# Icon: a correctly sized hicolor entry when we can resize, else the full-size PNG in
# /usr/share/pixmaps (both resolve Icon=dbaccounts from the .desktop file).
icon_src="build/appicon.png"
if python3 -c 'import PIL' >/dev/null 2>&1; then
  ICON_OUT="$work/$PKGNAME.png" ICON_SRC="$icon_src" python3 - <<'PY'
import os
from PIL import Image
Image.open(os.environ["ICON_SRC"]).convert("RGBA").resize((256, 256), Image.LANCZOS).save(os.environ["ICON_OUT"])
PY
  icon_target="/usr/share/icons/hicolor/256x256/apps/$PKGNAME.png"
else
  cp "$icon_src" "$work/$PKGNAME.png"
  icon_target="/usr/share/pixmaps/$PKGNAME.png"
fi

# fpm -t <format>: architecture and dependency names differ per distro family.
build_pkg() {
  local format="$1" arch="$2" file="$3"
  shift 3
  rm -f "$file"
  fpm -s dir -t "$format" \
    -n "$PKGNAME" -v "$VERSION" \
    --description "$DESCRIPTION" \
    --url "$URL" \
    --maintainer "$MAINTAINER" \
    --license "MIT" \
    --vendor "Michal Bartak" \
    --category "Development" \
    --architecture "$arch" \
    "$@" \
    --package "$file" \
    "$binary=/usr/bin/$OUTPUT" \
    "$work/$PKGNAME.desktop=/usr/share/applications/$PKGNAME.desktop" \
    "$work/$PKGNAME.png=$icon_target" > /dev/null
  echo "$file"
}

build_pkg deb amd64  "$DIST/$OUTPUT-v$VERSION-linux-$ARCH_LABEL.deb" \
  --depends libgtk-3-0 --depends libwebkit2gtk-4.1-0
build_pkg rpm x86_64 "$DIST/$OUTPUT-v$VERSION-linux-$ARCH_LABEL.rpm" \
  --depends gtk3 --depends webkit2gtk4.1
