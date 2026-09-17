#!/bin/sh
# Builds a universal (Apple Silicon + Intel) mic-off.app and zips it.
set -eu
cd "$(dirname "$0")/../.."
VERSION=${VERSION:-0.1.0}
LDFLAGS="-s -w -X main.version=$VERSION"
APP=dist/mic-off.app

rm -rf "$APP" dist/mic-off-macos.zip
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o dist/mic-off-arm64 .
if CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/mic-off-amd64 . 2>/dev/null; then
	lipo -create -output "$APP/Contents/MacOS/mic-off" dist/mic-off-arm64 dist/mic-off-amd64
	rm dist/mic-off-amd64
else
	echo "note: Intel build unavailable on this machine; shipping Apple Silicon only" >&2
	cp dist/mic-off-arm64 "$APP/Contents/MacOS/mic-off"
fi
rm dist/mic-off-arm64

sed "s/__VERSION__/$VERSION/g" packaging/macos/Info.plist > "$APP/Contents/Info.plist"
go run ./packaging/appicon -icns "$APP/Contents/Resources/AppIcon.icns"
# Ad-hoc signature so the binary runs on Apple Silicon. Replace "-" with a
# Developer ID and add notarization to avoid the Gatekeeper first-run prompt.
codesign --force --deep --sign "${CODESIGN_IDENTITY:--}" "$APP"

(cd dist && ditto -c -k --keepParent mic-off.app mic-off-macos.zip)
echo "built $APP and dist/mic-off-macos.zip"
