#!/usr/bin/env bash
set -euo pipefail

REPO="namlevia/Xiaozhi-Esp32-Server-Go-Vi"
INSTALL_DIR="${INSTALL_DIR:-$HOME/xiaozhi_server}"
API_URL="https://api.github.com/repos/${REPO}/releases/latest"

info() { printf '\033[1;34m[INFO]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[WARN]\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31m[ERROR]\033[0m %s\n' "$*" >&2; exit 1; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "Thiếu lệnh '$1'. Vui lòng cài đặt rồi chạy lại."
}

detect_platform() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m | tr '[:upper:]' '[:lower:]')"

  case "$os" in
    linux)
      case "$arch" in
        x86_64|amd64) echo "linux-amd64" ;;
        aarch64|arm64)
          if [ -n "${TERMUX_VERSION:-}" ] || [ -d /data/data/com.termux ]; then
            echo "android-arm64"
          else
            echo "linux-arm64"
          fi
          ;;
        armv7l|armv6l) echo "linux-arm" ;;
        *) fail "Chưa hỗ trợ Linux kiến trúc: $arch" ;;
      esac
      ;;
    darwin)
      case "$arch" in
        x86_64|amd64) echo "macos-amd64" ;;
        arm64|aarch64) echo "macos-arm64" ;;
        *) fail "Chưa hỗ trợ macOS kiến trúc: $arch" ;;
      esac
      ;;
    *) fail "Chưa hỗ trợ hệ điều hành: $os" ;;
  esac
}

asset_pattern_for_platform() {
  case "$1" in
    linux-amd64) echo 'xiaozhi_server-linux-amd64-lite-.*\.zip' ;;
    macos-amd64) echo 'xiaozhi_server-macos-amd64-lite-.*\.zip' ;;
    macos-arm64) echo 'xiaozhi_server-macos-arm64-lite-.*\.zip' ;;
    android-arm64) echo 'xiaozhi_server-android-arm64-lite-.*\.zip' ;;
    linux-arm64|linux-arm)
      fail "Raspberry Pi/Linux ARM hiện chưa có gói release chính thức. Vui lòng build thủ công hoặc dùng máy Linux amd64/macOS."
      ;;
    *) fail "Nền tảng chưa hỗ trợ: $1" ;;
  esac
}

json_get_latest_tag() {
  python3 - "$1" <<'PY'
import json, sys
with open(sys.argv[1], 'r', encoding='utf-8') as f:
    print(json.load(f).get('tag_name', ''))
PY
}

json_find_asset_url() {
  python3 - "$1" "$2" <<'PY'
import json, re, sys
path, pattern = sys.argv[1], sys.argv[2]
rx = re.compile(pattern)
with open(path, 'r', encoding='utf-8') as f:
    data = json.load(f)
for asset in data.get('assets', []):
    name = asset.get('name', '')
    if rx.fullmatch(name):
        print(asset.get('browser_download_url', ''))
        sys.exit(0)
PY
}

extract_zip() {
  local zip_file="$1" dest="$2"
  if command -v unzip >/dev/null 2>&1; then
    unzip -q "$zip_file" -d "$dest"
  elif command -v python3 >/dev/null 2>&1; then
    python3 - "$zip_file" "$dest" <<'PY'
import sys, zipfile
with zipfile.ZipFile(sys.argv[1]) as z:
    z.extractall(sys.argv[2])
PY
  else
    fail "Thiếu unzip hoặc python3 để giải nén file ZIP."
  fi
}

find_server_binary() {
  local root="$1"
  find "$root" -type f -name xiaozhi_server -print -quit
}

main() {
  need_cmd curl
  need_cmd python3

  local platform pattern tmp_dir release_json tag asset_url zip_file server_bin server_dir
  platform="$(detect_platform)"
  pattern="$(asset_pattern_for_platform "$platform")"

  info "Phát hiện nền tảng: $platform"
  info "Lấy thông tin release mới nhất từ ${REPO}"

  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' EXIT
  release_json="$tmp_dir/latest.json"
  curl -fsSL "$API_URL" -o "$release_json"

  tag="$(json_get_latest_tag "$release_json")"
  [ -n "$tag" ] || fail "Không đọc được tag release mới nhất."

  asset_url="$(json_find_asset_url "$release_json" "$pattern")"
  [ -n "$asset_url" ] || fail "Không tìm thấy gói Lite phù hợp cho $platform trong release $tag."

  info "Release mới nhất: $tag"
  info "Tải gói: $(basename "$asset_url")"

  mkdir -p "$INSTALL_DIR"
  zip_file="$tmp_dir/package.zip"
  curl -fL "$asset_url" -o "$zip_file"

  info "Giải nén vào: $INSTALL_DIR"
  rm -rf "$INSTALL_DIR/.extract"
  mkdir -p "$INSTALL_DIR/.extract"
  extract_zip "$zip_file" "$INSTALL_DIR/.extract"

  server_bin="$(find_server_binary "$INSTALL_DIR/.extract")"
  [ -n "$server_bin" ] || fail "Không tìm thấy file xiaozhi_server sau khi giải nén."

  server_dir="$(dirname "$server_bin")"
  rm -rf "$INSTALL_DIR/current"
  mv "$server_dir" "$INSTALL_DIR/current"
  chmod +x "$INSTALL_DIR/current/xiaozhi_server"
  mkdir -p "$INSTALL_DIR/current/logs" "$INSTALL_DIR/current/data" "$INSTALL_DIR/current/storage"

  cat <<EOF

Cài đặt xong.

Thư mục: $INSTALL_DIR/current
Chạy server:
  cd "$INSTALL_DIR/current"
  ./xiaozhi_server

Chạy nền và ghi log:
  cd "$INSTALL_DIR/current"
  nohup ./xiaozhi_server > logs/run.log 2>&1 &

Mở giao diện quản trị:
  http://127.0.0.1:8080

EOF
}

main "$@"
