#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

RAW=showcase/public/raw.mp4
OUT=assets/showcase.mp4
GIF=assets/showcase.gif

if [[ "${1:-}" != "--no-record" ]]; then
	go build -o gh-select .
	vhs scripts/demo.tape
fi

if [[ ! -d showcase/node_modules ]]; then
	(cd showcase && bun install)
fi
(cd showcase && bun run render)

ffmpeg -v error -y -i "$OUT" \
	-vf "fps=15,split[a][b];[a]palettegen=max_colors=128[p];[b][p]paletteuse=dither=bayer:bayer_scale=5" \
	"$GIF"

ls -la "$OUT" "$GIF"
