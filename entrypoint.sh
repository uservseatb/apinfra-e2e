#!/usr/bin/env bash
# Starts nested dockerd, then runs e2e tests against that daemon.
set -euo pipefail

DOCKER_STORAGE_DRIVER="${DOCKER_STORAGE_DRIVER:-vfs}"

dockerd \
	--host=unix:///var/run/docker.sock \
	--storage-driver="${DOCKER_STORAGE_DRIVER}" \
	>/var/log/dockerd.log 2>&1 &

echo "apinfra-e2e: waiting for nested dockerd (storage-driver=${DOCKER_STORAGE_DRIVER})..."
ready=0
for _ in $(seq 1 90); do
	if docker info >/dev/null 2>&1; then
		ready=1
		break
	fi
	sleep 1
done

if [[ "${ready}" -ne 1 ]]; then
	echo "apinfra-e2e: dockerd failed to become ready" >&2
	tail -n 200 /var/log/dockerd.log >&2 || true
	exit 1
fi

echo "apinfra-e2e: dockerd is ready"
docker version
docker compose version

if [[ ! -x /src/bin/apinfra ]]; then
	echo "apinfra-e2e: missing Linux binary at /src/bin/apinfra — run: task sync" >&2
	exit 1
fi

echo "apinfra-e2e: pre-pulling images used by e2e scenarios"
pull_with_retry() {
	local image="$1"
	local attempt
	for attempt in 1 2 3; do
		if docker pull "${image}"; then
			return 0
		fi
		echo "apinfra-e2e: docker pull ${image} failed (attempt ${attempt}/3); retrying..." >&2
		sleep $((attempt * 2))
	done
	echo "apinfra-e2e: docker pull ${image} failed after retries" >&2
	return 1
}
pull_with_retry alpine:latest

echo "apinfra-e2e: installing pebble (ACME stub for cert scenarios)"
go install github.com/letsencrypt/pebble/v2/cmd/pebble@v2.10.1
export PATH="$(go env GOPATH)/bin:${PATH}"
pebble -version || true

cd /src
# -parallel 1: testscript always calls t.Parallel(); share one Docker daemon serially.
exec go test -count=1 -parallel 1 -v ./...
