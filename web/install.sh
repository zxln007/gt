#!/bin/sh
# G-Tunnel one-line installer
#   curl -fsSL https://gtunnel.dev/install.sh | sh
# Environment overrides:
#   GT_REPO    GitHub repo to download from  (default: zxln007/gt, our fork)
#   GT_VERSION pin a release tag            (default: latest)
#   GT_BIN_DIR target directory             (default: /usr/local/bin if writable, else ~/.local/bin)
set -eu

REPO="${GT_REPO:-zxln007/gt}"
VERSION="${GT_VERSION:-latest}"
BIN_DIR="${GT_BIN_DIR:-}"

msg()  { printf '%s\n' "$1" >&2; }
die()  { msg "error: $1"; exit 1; }

# ── OS / arch detection ──────────────────────────────────────
os=$(uname -s)
case "$os" in
  Linux)  plat=linux ;;
  Darwin) plat=macos ;;
  MINGW*|MSYS*|CYGWIN*)
    msg "Windows detected: this installer targets POSIX shells."
    msg "Download the binary directly instead:"
    msg "  https://github.com/${REPO}/releases/latest/download/gt-win-x86_64.exe"
    exit 1 ;;
  *) die "unsupported operating system: ${os}" ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64)   arch=x86_64 ;;
  aarch64|arm64)  arch=aarch64 ;;
  riscv64)        arch=riscv64 ;;
  *) die "unsupported architecture: ${arch}" ;;
esac

asset="gt-${plat}-${arch}"
base="https://github.com/${REPO}/releases"
url="${base}/latest/download/${asset}"
[ "$VERSION" != "latest" ] && url="${base}/download/${VERSION}/${asset}"

# ── fetch helper (curl or wget) ──────────────────────────────
fetch() { # fetch <url> <output-file>
  if command -v curl >/dev/null 2>&1; then
    curl -fSL "$1" -o "$2"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$2" "$1"
  else
    die "neither curl nor wget is available"
  fi
}

tmp=$(mktemp -d) || die "cannot create temp dir"
trap 'rm -rf "$tmp"' EXIT INT TERM

# ── download ─────────────────────────────────────────────────
msg ">> G-Tunnel installer"
msg ">> repo: ${REPO}  version: ${VERSION}  asset: ${asset}"
msg ">> downloading ${url}"
fetch "$url" "${tmp}/${asset}" || die "download failed (asset may not exist for this platform)"

# ── verify: use checksums.txt when the release ships one ────
ck_url="${base}/latest/download/checksums.txt"
[ "$VERSION" != "latest" ] && ck_url="${base}/download/${VERSION}/checksums.txt"
if fetch "$ck_url" "${tmp}/checksums.txt" 2>/dev/null; then
  expected=$(awk -v f="$asset" '$2 == f {print $1}' "${tmp}/checksums.txt")
  if [ -n "$expected" ] && command -v sha256sum >/dev/null 2>&1; then
    printf '%s  %s\n' "$expected" "${tmp}/${asset}" | sha256sum -c - >/dev/null 2>&1 \
      || die "checksum mismatch, aborting (file may be corrupted or tampered)"
    msg ">> checksum ok (${expected})"
  fi
else
  msg ">> note: release ships no checksums.txt, skipping verification"
fi

# ── install ──────────────────────────────────────────────────
if [ -z "$BIN_DIR" ]; then
  if [ -w /usr/local/bin ] 2>/dev/null; then
    BIN_DIR=/usr/local/bin
  else
    BIN_DIR="${HOME}/.local/bin"
  fi
fi
mkdir -p "$BIN_DIR" || die "cannot create ${BIN_DIR}"

dest="${BIN_DIR}/gt"
mv -f "${tmp}/${asset}" "$dest"
chmod +x "$dest"

msg ">> installed: ${dest}"
case ":${PATH}:" in
  *":${BIN_DIR}:"*) ;;
  *) msg ">> note: ${BIN_DIR} is not in your PATH; add it with:"
     msg "      export PATH=\"${BIN_DIR}:\$PATH\"" ;;
esac
msg ">> next: gt --help"
