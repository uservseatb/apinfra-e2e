# AGENTS.md — apinfra-e2e

Instructions for AI coding agents working in this **end-to-end** test repository for [apinfra](../apinfra).

## What this repo is

Standalone Go module (`apinfra_e2e`) that drives the compiled `apinfra` binary through [testscript](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript) scenarios under `testdata/`.

It does **not** contain the agent source. Product code lives in `../apinfra`. This suite exercises real Git, Docker, Nginx (via the agent), and ACME (Pebble / local 429 stub).

`README.md` is the **product** README (synced from `../apinfra` by `task sync`). Treat it as the behavior contract when writing or planning scenarios. The E2E scenario inventory lives in `e2e-scenarios.md` and is appended to `README.md` by `task sync`.

## Prerequisites

1. Docker available to the test process (`DOCKER_*` env is propagated).
2. Compiled binary at `bin/apinfra` (required; Setup fails if missing).
3. Optional: Pebble on `PATH` / `$GOPATH/bin` — ACME scenarios install `github.com/letsencrypt/pebble/v2/cmd/pebble@v2.10.1` if absent.

## Commands

Requires [go-task](https://taskfile.dev) (`brew install go-task`) and Docker. Run from this repository root.

E2E runs inside **privileged Ubuntu DinD** (`run.sh` → nested `dockerd`) so host `:80`/`:443` do not collide with `apinfra-proxy`.

```bash
task sync    # GOOS=linux cross-build ../apinfra → bin/apinfra + README.md
task e2e     # build DinD image, run privileged container, go test inside
task test    # alias for e2e
task check   # fmt + vet + e2e
task fmt
task vet
```

`task sync` must produce a **Linux** `bin/apinfra` (the DinD container cannot exec a Darwin binary). Override sibling path: `task sync APINFRA_DIR=/path/to/apinfra`.

Inside DinD, tests use `-count=1 -parallel 1` (shared nested Docker + fixed ACME ports: pebble `:14000`, rate-limit stub `:18080`).

Do not commit `bin/apinfra` (see `.gitignore`). Do not vendor.

## Layout

```text
main_test.go          # testscript.Run + PATH/HOME setup
postinst_setup.go     # mirrors .deb postinst home layout under $WORK
docker_assert.go      # docker-assert-* custom commands
pebble.go             # pebble-start / pebble-stop
acme_ratelimit.go     # acme-ratelimit-start / stop (always-429 ACME stub)
testdata/*.txt        # one scenario per txtar archive
fixtures/             # pebble EAB config, minica PEMs, deb examples
bin/apinfra           # Linux binary under test (not in git; from task sync)
Dockerfile            # Ubuntu + Docker CE + Go image for DinD
entrypoint.sh         # nested dockerd → go test
run.sh                # docker build + privileged docker run
Taskfile.yml          # sync / e2e / check
```

## Writing a scenario

Each `testdata/<name>.txt` is a testscript archive:

1. Start with a `# Scenario: …` comment stating intent and assertions.
2. Init a local Git config repo under `git-repo/`, commit fixtures, point `.apinfra` at `file://$WORK/git-repo`.
3. Run `exec apinfra` (binary resolved from `bin/` via PATH). Soft-fail cycles still print `done duration=`; assert on `stderr` / `stdout` as the binary actually logs.
4. Assert Docker / files with built-ins (`exists`, `grep`, `stderr`, …) and custom cmds:
   - `docker-assert-running <name-substring>`
   - `docker-assert-no-public-ports <name-substring>`
   - `docker-assert-port-binding <name-substring> <hostIP> <hostPort> <containerPort>`
   - `pebble-start` / `pebble-stop`
   - `acme-ratelimit-start` / `acme-ratelimit-stop`
5. Embed host + Git fixtures after `-- path --` markers (txtar). Override `.apinfra` / `.env` in the archive when postinst defaults are wrong — Setup installs examples only if missing.

### Conventions

- **HOME = WorkDir** — agent host state (`.apinfra`, `.env`, `infra/`, `.retries`, `.events`, …) is per-scenario.
- Prefer **environment stubs** over Go mocks: local `file://` Git, fake `bin/dig` on `PATH`, Pebble, ACME 429 stub.
- Keep scenarios focused (one behavior). Reuse the alpine/`nc` HTTP stub pattern from existing files.
- ACME domain scenarios need DNS preflight: ship a tiny `bin/dig` stub and `chmod 0755` + prepend `PATH`.
- Always stop Pebble / rate-limit servers started in the script (`pebble-stop`, `acme-ratelimit-stop`).
- Match product behavior from `README.md` (validation soft-fail, cert rollback, port strip/rewrite, events). Do not assert implementation details that the README does not promise.
- Change only what the task needs; no drive-by refactors of shared harness code unless required by the new scenario.

## Agent behavior

- After harness Go changes: `go fmt ./...`, then `go test -count=1 -parallel 1 ./...` when Docker is available.
- Never commit secrets, real registry tokens, or live ACME credentials — fixtures use local Pebble EAB / stub directories only.
- Do not add Markdown docs the user did not ask for (except updating the README scenario list when adding/planning scenarios).
