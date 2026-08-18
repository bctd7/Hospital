#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 /path/to/hospital-images.tar.gz" >&2
  exit 1
fi

archive="$1"
test -f "${archive}" || {
  echo "Image archive not found: ${archive}" >&2
  exit 1
}

gzip -dc "${archive}" | docker load

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
HOSPITAL_PRELOADED_IMAGES=1 "${script_dir}/deploy.sh"
