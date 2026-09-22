#!/usr/bin/env bash
set -euo pipefail

MODULE="github.com/Abubakarr99/kresource"
BINARY_NAME="kubectl-resource"
VERSION="${VERSION:-latest}"

if ! command -v go >/dev/null 2>&1; then
  echo "error: go is not installed or not on PATH." >&2
  echo "       install it from https://go.dev/dl/ and try again." >&2
  exit 1
fi

# `go install` names the binary after the module's own last path element
# (kresource), not this plugin's kubectl-resource naming convention, so it's
# installed to a scratch GOBIN first and renamed into place below.
tmp_gobin="$(mktemp -d)"
trap 'rm -rf "${tmp_gobin}"' EXIT

echo "==> Installing ${MODULE}@${VERSION} ($(go env GOOS)/$(go env GOARCH))"
GOBIN="${tmp_gobin}" go install "${MODULE}@${VERSION}"

# Prefer a directory the user can already write to over /usr/local/bin, so
# this doesn't demand sudo on machines where ~/.local/bin (or an equivalent
# user bin dir) is already on PATH.
resolve_install_dir() {
  if [ -n "${INSTALL_DIR:-}" ]; then
    printf '%s\n' "${INSTALL_DIR}"
    return
  fi
  case ":${PATH}:" in
    *":${HOME}/.local/bin:"*) printf '%s\n' "${HOME}/.local/bin"; return ;;
    *":${HOME}/bin:"*) printf '%s\n' "${HOME}/bin"; return ;;
  esac
  printf '%s\n' "/usr/local/bin"
}

install_dir="$(resolve_install_dir)"
mkdir -p "${install_dir}" 2>/dev/null || true

echo "==> Installing to ${install_dir}/${BINARY_NAME}"
if [ -w "${install_dir}" ]; then
  mv "${tmp_gobin}/kresource" "${install_dir}/${BINARY_NAME}"
else
  echo "==> ${install_dir} isn't writable, using sudo"
  sudo mv "${tmp_gobin}/kresource" "${install_dir}/${BINARY_NAME}"
fi
chmod +x "${install_dir}/${BINARY_NAME}"

case ":${PATH}:" in
  *":${install_dir}:"*) ;;
  *)
    echo
    echo "warning: ${install_dir} is not on your PATH, so kubectl won't find this plugin yet."
    echo "  add it, e.g.: export PATH=\"${install_dir}:\$PATH\""
    ;;
esac

echo
echo "==> Installed. Try: kubectl resource list"
