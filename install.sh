#!/bin/sh
set -e

GH_REPO="loops-so/cli"
GH_ASSETS_URL="https://github.com/${GH_REPO}/releases/download"

#
# functions adapted from https://github.com/client9/shlib
#
echoerr() {
  echo "$@" 1>&2
}

github_release() {
  owner_repo=$1
  version=$2
  test -z "$version" && version="latest"
  if [ "$version" = "latest" ]; then
    giturl="https://api.github.com/repos/${owner_repo}/releases/latest"
  else
    giturl="https://api.github.com/repos/${owner_repo}/releases/tags/${version}"
  fi
  json=$(http_copy "$giturl")
  test -z "$json" && return 1
  version=$(echo "$json" | tr -s '\n' ' ' | sed 's/.*"tag_name": *"//' | sed 's/".*//')
  test -z "$version" && return 1
  echo "$version"
}

http_download_curl() {
  local_file=$1
  source_url=$2
  code=$(curl -w '%{http_code}' -sL --proto '=https' --tlsv1.2 -o "$local_file" "$source_url")
  if [ "$code" != "200" ]; then
    log_debug "http_download_curl received HTTP status $code"
    return 1
  fi
  return 0
}

http_download_wget() {
  local_file=$1
  source_url=$2
  wget -q -O "$local_file" "$source_url"
}

http_download() {
  log_debug "http_download $2"
  if is_command curl; then
    http_download_curl "$@"
    return
  elif is_command wget; then
    http_download_wget "$@"
    return
  fi
  log_crit "http_download unable to find wget or curl"
  return 1
}

http_copy() {
  tmp=$(mktemp)
  http_download "${tmp}" "$1" || return 1
  body=$(cat "$tmp")
  rm -f "${tmp}"
  echo "$body"
}

is_command() {
  command -v "$1" >/dev/null
}

log_prefix() {
  echo "$0"
}

# default priority
_logp=6

log_set_priority() {
  _logp="$1"
}

log_priority() {
  if test -z "$1"; then
    echo "$_logp"
    return
  fi
  [ "$1" -le "$_logp" ]
}

log_tag() {
  case $1 in
    0) echo "emerg" ;;
    1) echo "alert" ;;
    2) echo "crit" ;;
    3) echo "err" ;;
    4) echo "warning" ;;
    5) echo "notice" ;;
    6) echo "info" ;;
    7) echo "debug" ;;
    *) echo "$1" ;;
  esac
}

log_debug() {
  log_priority 7 || return 0
  echoerr "$(log_prefix)" "$(log_tag 7)" "$@"
}

log_info() {
  log_priority 6 || return 0
  echoerr "$(log_prefix)" "$(log_tag 6)" "$@"
}

log_err() {
  log_priority 3 || return 0
  echoerr "$(log_prefix)" "$(log_tag 3)" "$@"
}

log_crit() {
  log_priority 2 || return 0
  echoerr "$(log_prefix)" "$(log_tag 2)" "$@"
}

uname_arch() {
  arch=$(uname -m)
  case $arch in
    x86) arch="i386" ;;
    i686) arch="i386" ;;
    aarch64) arch="arm64" ;;
    armv5*) arch="armv5" ;;
    armv6*) arch="armv6" ;;
    armv7*) arch="armv7" ;;
  esac
  echo "${arch}"
}

uname_os() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')

  case "$os" in
    msys*) os="windows" ;;
    mingw*) os="windows" ;;
    cygwin*) os="windows" ;;
    win*) os="windows" ;;
  esac

  echo "$os"
}

hash_sha256() {
  TARGET=${1:-/dev/stdin}
  if is_command gsha256sum; then
    hash=$(gsha256sum "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command sha256sum; then
    hash=$(sha256sum "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command shasum; then
    hash=$(shasum -a 256 "$TARGET" 2>/dev/null) || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command openssl; then
    hash=$(openssl dgst -sha256 "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 2
  else
    log_crit "hash_sha256 unable to find command to compute sha-256 hash"
    return 1
  fi
}

hash_sha256_verify() {
  TARGET=$1
  checksums=$2
  if [ -z "$checksums" ]; then
    log_err "hash_sha256_verify checksum file not specified in arg2"
    return 1
  fi
  BASENAME=${TARGET##*/}
  want=$(grep "${BASENAME}" "${checksums}" 2>/dev/null | tr '\\t' ' ' | cut -d ' ' -f 1)
  if [ -z "$want" ]; then
    log_err "hash_sha256_verify unable to find checksum for '${TARGET}' in '${checksums}'"
    return 1
  fi
  got=$(hash_sha256 "$TARGET")
  if [ "$want" != "$got" ]; then
    log_err "hash_sha256_verify checksum for '$TARGET' did not verify ${want} vs $got"
    return 1
  fi
}

untar() {
  tarball=$1
  case "${tarball}" in
    *.tar.gz | *.tgz) tar -xzf "${tarball}" ;;
    *.tar) tar -xf "${tarball}" ;;
    *.zip) unzip "${tarball}" ;;
    *)
      log_err "untar unknown archive format for ${tarball}"
      return 1
      ;;
  esac
}
#
# end functions from https://github.com/client9/shlib
#

OS=$(uname_os)
ARCH=$(uname_arch)

TAG="${1-latest}"
INSTALL_DIR="${2-$HOME/.local/bin}"
PROJ_NAME="loops_cli"
SHORT_BIN_NAME="loops"
GH_RELEASE=$(github_release "$GH_REPO" "$TAG")
if [ "$OS" = "windows" ]; then
  GH_RELEASE_FILENAME="${PROJ_NAME}_${OS}_${ARCH}.zip"
else
  GH_RELEASE_FILENAME="${PROJ_NAME}_${OS}_${ARCH}.tar.gz"
fi
DOWNLOAD_URL="${GH_ASSETS_URL}/${GH_RELEASE}/${GH_RELEASE_FILENAME}"
VERSION_NO_V=$(echo "$GH_RELEASE" | sed 's/^v//')
CHECKSUMS_FILENAME="${PROJ_NAME}_${VERSION_NO_V}_checksums.txt"
CHECKSUMS_URL="${GH_ASSETS_URL}/${GH_RELEASE}/${CHECKSUMS_FILENAME}"

execute() {
  TMPDIR=$(mktemp -d)
  if ! http_download "${TMPDIR}/${GH_RELEASE_FILENAME}" "$DOWNLOAD_URL"; then
    log_err "Failed to download $DOWNLOAD_URL"
    return 1
  fi
  if ! http_download "${TMPDIR}/${CHECKSUMS_FILENAME}" "$CHECKSUMS_URL"; then
    log_err "Failed to download checksums from $CHECKSUMS_URL"
    return 1
  fi
  hash_sha256_verify "${TMPDIR}/${GH_RELEASE_FILENAME}" "${TMPDIR}/${CHECKSUMS_FILENAME}"
  (cd "$TMPDIR" && untar "$GH_RELEASE_FILENAME")
  mkdir -p "$INSTALL_DIR"

  if [ "$OS" = "windows" ]; then
    SHORT_BIN_NAME="${SHORT_BIN_NAME}.exe"
  fi
  install "${TMPDIR}/${SHORT_BIN_NAME}" "${INSTALL_DIR}/${SHORT_BIN_NAME}"

  rm -rf "$TMPDIR"
}

printf "Installing %s %s for %s %s... " "$PROJ_NAME" "$GH_RELEASE" "$OS" "$ARCH"
execute
echo "Done!"
printf '\033[32m✓\033[0m Installed to %s\n' "${INSTALL_DIR}/${SHORT_BIN_NAME}"
"${INSTALL_DIR}/${SHORT_BIN_NAME}" --version
