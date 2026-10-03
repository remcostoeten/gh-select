#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY="gh-select"
PREFIX="${PREFIX:-$HOME/.local}"
BIN_DIR="$PREFIX/bin"

cd "$ROOT"

function usage() {
	cat <<EOF
Usage: scripts/dev.sh <command>

Commands:
  build          Build ./$BINARY with the version from git describe
  run [args]     Build, then run ./$BINARY with the given arguments
  test           Run go test ./...
  vet            Run go vet ./...
  check          Run vet, test and build
  install        Build and install as a local gh extension (gh select)
  install-bin    Build and copy the binary to \$PREFIX/bin (default ~/.local/bin)
  uninstall      Remove the gh extension and the binary in \$PREFIX/bin
  clean          Remove the built binary
  help           Show this help (-h, --h, --help)

Environment:
  PREFIX         Install prefix for install-bin and uninstall (current: $PREFIX)
EOF
}

function info() {
	printf '\033[1;34m==>\033[0m %s\n' "$1"
}

function fail() {
	printf '\033[1;31merror:\033[0m %s\n' "$1" >&2
	exit 1
}

function require() {
	command -v "$1" >/dev/null 2>&1 || fail "$1 is not installed"
}

function version() {
	git describe --tags --always --dirty 2>/dev/null || echo "dev"
}

function build() {
	require go
	local ver
	ver="$(version)"
	info "Building $BINARY $ver"
	go build -ldflags "-s -w -X main.version=$ver" -o "$BINARY" .
}

function run_tests() {
	require go
	info "Running tests"
	go test ./...
}

function vet() {
	require go
	info "Running go vet"
	go vet ./...
}

function extension_installed() {
	gh extension list 2>/dev/null | grep -q "^gh ${BINARY#gh-}\b"
}

function install_extension() {
	require gh
	build
	if extension_installed; then
		info "Removing existing $BINARY extension"
		gh extension remove "$BINARY"
	fi
	info "Installing local extension from $ROOT"
	gh extension install .
	info "Installed, run with: gh select"
}

function install_bin() {
	build
	mkdir -p "$BIN_DIR"
	install -m 0755 "$BINARY" "$BIN_DIR/$BINARY"
	info "Installed to $BIN_DIR/$BINARY"
	case ":$PATH:" in
	*":$BIN_DIR:"*) ;;
	*) info "$BIN_DIR is not on your PATH" ;;
	esac
}

function uninstall() {
	local removed=0
	if command -v gh >/dev/null 2>&1 && extension_installed; then
		info "Removing gh extension $BINARY"
		gh extension remove "$BINARY"
		removed=1
	fi
	if [[ -f "$BIN_DIR/$BINARY" ]]; then
		info "Removing $BIN_DIR/$BINARY"
		rm -f "$BIN_DIR/$BINARY"
		removed=1
	fi
	if [[ $removed -eq 0 ]]; then
		info "Nothing to uninstall"
	fi
}

function clean() {
	info "Removing ./$BINARY"
	rm -f "$BINARY"
}

function main() {
	local command="${1:-help}"
	shift || true

	case "$command" in
	build) build ;;
	run)
		build
		"./$BINARY" "$@"
		;;
	test) run_tests ;;
	vet) vet ;;
	check)
		vet
		run_tests
		build
		;;
	install) install_extension ;;
	install-bin) install_bin ;;
	uninstall) uninstall ;;
	clean) clean ;;
	help | -h | --h | --help) usage ;;
	*)
		usage >&2
		fail "unknown command: $command"
		;;
	esac
}

main "$@"
