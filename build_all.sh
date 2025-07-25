#!/bin/bash
set -e

go mod download

mkdir -p bin

for d in ./cmd/*; do
  if [ -d "$d" ] && [ -f "$d/main.go" ]; then
    name=$(basename "$d")
    echo "Building $name..."
    go build -o "bin/$name" "$d/main.go"
  fi
done

echo "All binaries built in ./bin/" 