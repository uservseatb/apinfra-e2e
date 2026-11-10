# apinfra-e2e

[![Go Version](https://img.shields.io/github/go-mod/go-version/uservseatb/apinfra-e2e)](https://github.com/uservseatb/apinfra-e2e/blob/main/go.mod)
[![License](https://img.shields.io/github/license/uservseatb/apinfra-e2e)](LICENSE)
![Testing - E2E](https://img.shields.io/badge/Testing-E2E-blue)
![Framework - testscript](https://img.shields.io/badge/Framework-testscript-blue)
![Environment - DinD](https://img.shields.io/badge/Environment-DinD-blue)
![ACME Stub - Pebble](https://img.shields.io/badge/ACME%20Stub-Pebble-brightgreen)

End-to-end tests for [apinfra](https://github.com/uservseatb/apinfra) — a GitOps agent that deploys docker-compose apps with Nginx and SSL certificates.

This repo does **not** contain the agent source. It runs the compiled `apinfra` binary against real Git, Docker, Nginx, and local ACME stubs (Pebble / rate-limit), using [testscript](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript) scenarios.

Licensed under the MIT License — see [LICENSE](LICENSE).

## What it covers

Scenarios live under `testdata/` and are grouped by theme:

| Theme | Focus |
|-------|--------|
| `cycle/` | First deploy, idempotent re-run, validation soft-fail |
| `docker/` | Compose ports, `network_mode`, heal, enable/disable projects |
| `nginx/` | Proxy conf write / rollback |
| `certs/` | ACME issue, fallback, downgrade, retries, DNS preflight |

Full inventory: [`E2E_SCENARIOS.md`](E2E_SCENARIOS.md).  
Product behavior contract (synced from apinfra): [`testdata/README.md`](testdata/README.md).

## Prerequisites

1. [Docker](https://docs.docker.com/get-docker/) and [go-task](https://taskfile.dev) (`brew install go-task`)
2. A **Linux** `apinfra` binary at `bin/apinfra` (DinD cannot run a Darwin binary — build or copy it from `../apinfra` first)

## Run

Tests run inside a privileged Ubuntu Docker-in-Docker container so host ports `:80` / `:443` do not collide with `apinfra-proxy`.

```bash
task e2e     # build DinD image, run tests inside
task fmt
task vet

# One theme: cycle | docker | nginx | certs | env
task e2e -- docker

# One scenario: testdata/<theme>/<name>.txt
task e2e -- cycle/happy_path
```

Or directly:

```bash
./env/run.sh
./env/run.sh docker
./env/run.sh cycle/happy_path
```

## Layout

```text
e2e/              TestE2e<Theme> entrypoints (package e2e_tests)
env/              Test helpers + DinD runner (run.sh, Dockerfile)
testdata/         Scenarios (*.txt) + product README
fixtures/         Pebble / ACME / deb examples
bin/apinfra       Binary under test (not committed)
E2E_SCENARIOS.md  Scenario inventory
AGENTS.md         Notes for AI coding agents
```
