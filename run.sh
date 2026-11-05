#!/usr/bin/env bash
# Local e2e runner via Docker-in-Docker (DinD).
#
# Why DinD on Mac / busy Linux hosts:
#   apinfra-proxy publishes host ports 80 and 443. If something else already
#   binds :80 on the machine (local demos, another proxy), compose up fails and
#   app containers never start. Nesting a dockerd inside a privileged Ubuntu
#   container gives the test its own free :80/:443 — host listeners do not matter.
#
# Prerequisites:
#   task sync   # Linux apinfra binary into bin/ (GOOS=linux cross-compile)
#
# Usage (from repo root, or via: task e2e):
#   ./run.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "${ROOT}"

if [[ ! -f bin/apinfra ]]; then
	echo "apinfra-e2e: bin/apinfra not found — run: task sync" >&2
	exit 1
fi

IMAGE_NAME="${APINFRA_E2E_IMAGE:-apinfra-e2e:local}"
GO_VERSION="$(sed -n 's/^go //p' go.mod | head -n1 | tr -d '[:space:]')"
if [[ -z "${GO_VERSION}" ]]; then
	echo "apinfra-e2e: failed to read go version from go.mod" >&2
	exit 1
fi

case "$(uname -m)" in
	x86_64) TARGETARCH=amd64 ;;
	aarch64|arm64) TARGETARCH=arm64 ;;
	*)
		echo "apinfra-e2e: unsupported architecture: $(uname -m)" >&2
		exit 1
		;;
esac

echo "apinfra-e2e: building ${IMAGE_NAME} (GO_VERSION=${GO_VERSION} TARGETARCH=${TARGETARCH})"
docker build \
	--build-arg "GO_VERSION=${GO_VERSION}" \
	--build-arg "TARGETARCH=${TARGETARCH}" \
	-f Dockerfile \
	-t "${IMAGE_NAME}" \
	.

# Do not publish 80/443 to the Mac/Linux host — they stay inside nested dockerd.
# Source mount keeps iteration fast without rebuilding the image for code changes.
echo "apinfra-e2e: running privileged DinD container"
docker run --rm --privileged \
	--name "apinfra-e2e-$$" \
	-v "${ROOT}:/src" \
	-e "DOCKER_STORAGE_DRIVER=${DOCKER_STORAGE_DRIVER:-vfs}" \
	"${IMAGE_NAME}"
