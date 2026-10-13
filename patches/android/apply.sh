#!/bin/sh
set -eu
cd "$(dirname "$0")/../.."
module=gioui.org
version=$(sed -n 's/^[[:space:]]*gioui.org \(v[^[:space:]]*\).*/\1/p' go.mod | head -n 1)
(cd /tmp && GOFLAGS=-mod=mod go mod download "$module@$version" gioui.org/cmd@v0.10.3)
cache=$(go env GOMODCACHE)
rm -rf third_party/gio third_party/gogio
cp -R "$cache/$module@$version" third_party/gio
cp -R "$cache/gioui.org/cmd@v0.10.3" third_party/gogio
chmod -R u+w third_party/gio third_party/gogio
patch -s -p1 -d third_party/gio < patches/android/gio-activity.patch
cp patches/android/*.java third_party/gio/app/
patch -s -p1 -d third_party/gogio < patches/android/gogio.patch
