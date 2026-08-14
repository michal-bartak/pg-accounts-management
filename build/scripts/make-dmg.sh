#!/usr/bin/env bash
# Build a macOS .dmg installer from the app bundle in build/bin.
#
#   OUTPUT=pgCowboy VERSION=0.3.0 ARCH_LABEL=arm64 build/scripts/make-dmg.sh
#
# Steps: rebuild the app icon as a full multi-size .icns, ad-hoc re-sign the bundle,
# then stage <app> + an /Applications symlink into a compressed DMG with a drag-here
# background. Run from the repo root; writes dist/<OUTPUT>-v<VERSION>-macos-<ARCH>.dmg.
set -euo pipefail

OUTPUT="${OUTPUT:-pgCowboy}"
VERSION="${VERSION:-$(tr -d ' \n\r' < VERSION)}"
ARCH_LABEL="${ARCH_LABEL:-$(uname -m)}"
DIST="${DIST:-dist}"
VOLNAME="${VOLNAME:-$OUTPUT}"

app="$(find build/bin -maxdepth 1 -name '*.app' -type d | head -1)"
[ -n "$app" ] || { echo "ERROR: no .app bundle in build/bin — run 'make build' first"; exit 1; }
appname="$(basename "$app")"
dmg="$DIST/$OUTPUT-v$VERSION-macos-$ARCH_LABEL.dmg"
mkdir -p "$DIST"
rm -f "$dmg"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# 1. Wails writes a single-resolution icns; replace it with the full icon set so the
#    Dock, Finder and Get Info all render crisply.
if command -v iconutil >/dev/null 2>&1 && [ -f build/appicon.png ]; then
  iconset="$work/app.iconset"
  mkdir -p "$iconset"
  for spec in 16:icon_16x16 32:icon_16x16@2x 32:icon_32x32 64:icon_32x32@2x \
              128:icon_128x128 256:icon_128x128@2x 256:icon_256x256 512:icon_256x256@2x \
              512:icon_512x512; do
    px="${spec%%:*}"; name="${spec##*:}"
    sips -z "$px" "$px" build/appicon.png --out "$iconset/$name.png" > /dev/null
  done
  cp build/appicon.png "$iconset/icon_512x512@2x.png"
  iconutil -c icns "$iconset" -o "$app/Contents/Resources/iconfile.icns"
  # Re-sign the outer bundle only — --deep strips the nested binary's signature.
  codesign --force --sign - "$app"
fi

# 2. Drag-here background (brand blue + arrow). Optional: skipped without Pillow.
bg=""
if python3 -c 'import PIL' >/dev/null 2>&1; then
  bg="$work/background.png"
  BG_OUT="$bg" python3 - <<'PY'
import os
from PIL import Image, ImageDraw

W, H = 540, 440
BG = (222, 233, 252, 255)     # light tint of the app's --primary blue
ARROW = (58, 115, 232, 235)   # --primary (light theme)
img = Image.new("RGBA", (W, H), BG)
draw = ImageDraw.Draw(img)

TRI_W, TRI_H, TRI_GAP, TRI_N = 15, 17, 8, 5
row_w = TRI_N * TRI_W + (TRI_N - 1) * TRI_GAP
start_x = (209 + 331) / 2 - row_w / 2   # centred between the two icon positions
cy = 185
for i in range(TRI_N):
    x = start_x + i * (TRI_W + TRI_GAP)
    draw.polygon([(x, cy - TRI_H / 2), (x + TRI_W, cy), (x, cy + TRI_H / 2)], fill=ARROW)

img.convert("RGB").save(os.environ["BG_OUT"])
PY
fi

# 3. Stage the bundle, the /Applications drop target and the hidden background.
staging="$work/staging"
mkdir -p "$staging"
cp -R "$app" "$staging/"
ln -s /Applications "$staging/Applications"
if [ -n "$bg" ]; then
  mkdir "$staging/.background"
  cp "$bg" "$staging/.background/background.png"
fi

# 4. Writable image → set the Finder window layout → compress read-only.
rw="$work/rw.dmg"
hdiutil create -volname "$VOLNAME" -srcfolder "$staging" -ov -format UDRW -size 300m "$rw" > /dev/null

attach_log="$work/attach.txt"
hdiutil attach -readwrite -noverify "$rw" > "$attach_log"
device="$(awk '/^\/dev\/disk/ {print $1; exit}' "$attach_log")"
[ -n "$device" ] || { echo "ERROR: could not determine the mounted device"; exit 1; }
sleep 2

# Finder scripting needs Automation permission; a denial only costs us the icon layout.
if ! osascript \
  -e 'on run argv' \
  -e '  set volName to item 1 of argv' \
  -e '  set appName to item 2 of argv' \
  -e '  set useBg to item 3 of argv' \
  -e '  tell application "Finder"' \
  -e '    tell disk volName' \
  -e '      open' \
  -e '      set current view of container window to icon view' \
  -e '      set toolbar visible of container window to false' \
  -e '      set statusbar visible of container window to false' \
  -e '      set bounds of container window to {200, 100, 740, 540}' \
  -e '      set opts to icon view options of container window' \
  -e '      set arrangement of opts to not arranged' \
  -e '      set icon size of opts to 128' \
  -e '      if useBg is "yes" then set background picture of opts to file ".background:background.png"' \
  -e '      set position of item appName of container window to {135, 185}' \
  -e '      set position of item "Applications" of container window to {405, 185}' \
  -e '      close' \
  -e '      open' \
  -e '      update without registering applications' \
  -e '      delay 2' \
  -e '      close' \
  -e '    end tell' \
  -e '  end tell' \
  -e 'end run' \
  "$VOLNAME" "$appname" "$([ -n "$bg" ] && echo yes || echo no)" > /dev/null 2>&1; then
  echo "WARNING: could not set the DMG window layout (Finder automation unavailable)"
fi

sync
hdiutil detach "$device" -force > /dev/null
hdiutil convert "$rw" -format UDZO -imagekey zlib-level=9 -o "$dmg" > /dev/null

echo "$dmg"
