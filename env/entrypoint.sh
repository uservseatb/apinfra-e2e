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
# Extra args come from: task e2e -- ... / ./env/run.sh ...
# Shorthand (preferred):
#   task e2e -- docker
#   task e2e -- cycle/happy_path
# Still accepted: -run TestE2eDocker / -run TestScenario theme/name
theme_to_run() {
	case "$1" in
	cycle) echo TestE2eCycle ;;
	docker) echo TestE2eDocker ;;
	nginx) echo TestE2eNginx ;;
	certs|cert) echo TestE2eCerts ;;
	env) echo TestE2eEnv ;;
	*) return 1 ;;
	esac
}

go_args=()
theme=""
scenario=""
wants_scenario=0
has_run=0
i=1
while [[ "${i}" -le "$#" ]]; do
	arg="${!i}"
	case "${arg}" in
	-run)
		next="$((i + 1))"
		if [[ "${next}" -gt "$#" ]]; then
			echo "apinfra-e2e: -run requires an argument" >&2
			exit 1
		fi
		pattern="${!next}"
		go_args+=(-run "${pattern}")
		has_run=1
		if [[ "${pattern}" == *TestScenario* ]]; then
			wants_scenario=1
		fi
		i=$((i + 2))
		continue
		;;
	-run=*)
		pattern="${arg#-run=}"
		go_args+=("${arg}")
		has_run=1
		if [[ "${pattern}" == *TestScenario* ]]; then
			wants_scenario=1
		fi
		i=$((i + 1))
		continue
		;;
	-args)
		echo "apinfra-e2e: pass theme or theme/name after -- ; example: task e2e -- docker   or   task e2e -- cycle/happy_path" >&2
		exit 1
		;;
	esac

	if [[ "${arg}" == -* ]]; then
		go_args+=("${arg}")
		i=$((i + 1))
		continue
	fi

	path="${arg#./}"
	path="${path#testdata/}"

	# Bare theme: cycle|docker|nginx|certs (cert → certs)
	if run_name="$(theme_to_run "${path}")"; then
		if [[ -n "${theme}" ]]; then
			echo "apinfra-e2e: multiple themes: ${theme} and ${path}" >&2
			exit 1
		fi
		if [[ "${path}" == "cert" ]]; then
			theme="certs"
		else
			theme="${path}"
		fi
		i=$((i + 1))
		continue
	fi

	# Scenario: theme/name or theme/name.txt
	if [[ "${path}" =~ ^[A-Za-z0-9_-]+/[A-Za-z0-9_.-]+$ ]]; then
		if [[ -n "${scenario}" ]]; then
			echo "apinfra-e2e: multiple scenario paths: ${scenario} and ${path}" >&2
			exit 1
		fi
		scenario="${path}"
		i=$((i + 1))
		continue
	fi

	echo "apinfra-e2e: unknown arg ${arg@Q} (expected theme, theme/name, or go test flags)" >&2
	exit 1
done

if [[ -n "${theme}" && -n "${scenario}" ]]; then
	echo "apinfra-e2e: pass either a theme (${theme}) or a scenario (${scenario}), not both" >&2
	exit 1
fi
if [[ -n "${scenario}" ]]; then
	wants_scenario=1
fi
if [[ "${wants_scenario}" -eq 1 && -z "${scenario}" ]]; then
	echo "apinfra-e2e: scenario path required: task e2e -- theme/name" >&2
	exit 1
fi
if [[ "${has_run}" -eq 1 && ( -n "${theme}" || -n "${scenario}" ) ]]; then
	echo "apinfra-e2e: use either -run ... or a theme/scenario shorthand, not both" >&2
	exit 1
fi

if [[ -n "${theme}" ]]; then
	run_name="$(theme_to_run "${theme}")"
	go_args+=(-run "${run_name}")
	echo "apinfra-e2e: running theme ${theme} (${run_name})"
fi

if [[ -n "${scenario}" ]]; then
	script="${scenario}"
	[[ "${script}" == *.txt ]] || script="${script}.txt"
	if [[ ! -f "testdata/${script}" ]]; then
		echo "apinfra-e2e: scenario not found: testdata/${script}" >&2
		exit 1
	fi
	go_args+=(-run TestScenario -args "${scenario}")
	echo "apinfra-e2e: running scenario testdata/${script}"
fi

exec go test -count=1 -parallel 1 -v ./... "${go_args[@]}"
