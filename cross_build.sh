#!/usr/bin/env bash

OSLIST="linux windows darwin"
ARCHLIST="amd64 arm arm64"
APPNAME="nin"

for os in ${OSLIST}; do
  for arch in ${ARCHLIST}; do
    if [[ "$os/$arch" =~ ^(windows/arm64|darwin/arm)$ ]]; then continue; fi

    echo "Building binary for $os $arch"
    mkdir -p releases/${os}/${arch}

    # Windows binaries need .exe suffix
    ext=""
    if [ "$os" == "windows" ]; then
      ext=".exe"
    fi

    CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build \
      -o releases/${os}/${arch}/${APPNAME}-${os}-${arch}${ext}
  done
done
