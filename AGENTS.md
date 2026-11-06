# AGENTS.md — apinfra-e2e

Instructions for AI coding agents working in this **end-to-end** test repository for [apinfra](../apinfra).

## What this repo is

Standalone Go module (`apinfra_e2e`) that drives the compiled `apinfra` binary through [testscript](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript) scenarios under `testdata/`.

It does **not** contain the agent source. Product code lives in `../apinfra`. This suite exercises real Git, Docker, Nginx (via the agent), and ACME (Pebble / local 429 stub).

`README.md` is the **product** README (kept in sync with `../apinfra`). Treat it as the behavior contract when writing or planning scenarios. The inventory of implemented E2E scenarios lives only in [`E2E_SCENARIOS.md`](E2E_SCENARIOS.md) — never in `README.md`.

## Prerequisites

1. Docker available to the test process (`DOCKER_*` env is propagated).
2. Compiled **Linux** binary at `bin/apinfra` (required; Setup fails if missing). The DinD container cannot exec a Darwin binary — build/copy it from `../apinfra` yourself before `task e2e`.
3. Optional: Pebble on `PATH` / `$GOPATH/bin` — ACME scenarios install `github.com/letsencrypt/pebble/v2/cmd/pebble@v2.10.1` if absent.

## Commands

Requires [go-task](https://taskfile.dev) (`brew install go-task`) and Docker. Run from this repository root.

E2E runs inside **privileged Ubuntu DinD** (`run.sh` → nested `dockerd`) so host `:80`/`:443` do not collide with `apinfra-proxy`.

```bash
task e2e     # build DinD image, run privileged container, go test inside
task test    # alias for e2e
task check   # fmt + vet + e2e
task fmt
task vet
```

Inside DinD, tests use `-count=1 -parallel 1` (shared nested Docker + fixed ACME ports: pebble `:14000`, rate-limit stub `:18080`).

Do not commit `bin/apinfra` (see `.gitignore`). Do not vendor.

## Layout

```text
main_test.go          # TestE2e<Theme> per testdata/<theme>/ + PATH/HOME setup
postinst_setup.go     # mirrors .deb postinst home layout under $WORK
docker_assert.go      # docker-assert-* custom commands
pebble.go             # pebble-start / pebble-stop
acme_ratelimit.go     # acme-ratelimit-start / stop (always-429 ACME stub)
testdata/<theme>/     # theme folders (one level); one scenario per *.txt archive
E2E_SCENARIOS.md      # inventory of implemented scenarios (source of truth)
fixtures/             # pebble EAB config, minica PEMs, deb examples
bin/apinfra           # Linux binary under test (not in git; supply before e2e)
Dockerfile            # Ubuntu + Docker CE + Go image for DinD
entrypoint.sh         # nested dockerd → go test
run.sh                # docker build + privileged docker run
Taskfile.yml          # e2e / check / fmt / vet
```

## Scenario layout (`testdata/`)

Scenarios live in **theme subdirectories** — exactly one level: `testdata/<theme>/<name>.txt`.

- **No** scripts at `testdata/` root.
- **No** nested theme folders (`testdata/a/b/…`); only one level of themes.
- Prefer a **small set of broad themes** over many tiny folders. Reuse an existing theme when the behavior fits; add a new theme only when a cluster of scenarios would clearly belong together.
- Each theme has a dedicated `TestE2e<Theme>` in `main_test.go` with `Dir: "testdata/<theme>"`. Current themes:

| Theme | Test | Focus |
|-------|------|--------|
| `cycle/` | `TestE2eCycle` | Agent cycle: first deploy, idempotent re-run, validation soft-fail |
| `docker/` | `TestE2eDocker` | Compose/Docker apply: ports, `network_mode`, heal, enable/disable projects |
| `nginx/` | `TestE2eNginx` | Nginx proxy conf write / rollback |
| `certs/` | `TestE2eCerts` | Certificate issuance: ACME, fallback, downgrade, retries, DNS preflight |

When adding a scenario, place it under the matching theme. If none fit and the new area is substantial, introduce a new top-level theme, add `TestE2e<Theme>` in `main_test.go`, and document it here and in `E2E_SCENARIOS.md` in the same change.

## Scenario inventory & workflow

**`README.md` is never edited for scenarios.** Product README mirrors `../apinfra`. Do not append scenario lists, planned gaps, or e2e notes to it.

**`E2E_SCENARIOS.md` is the only inventory.** Keep every implemented `testdata/<theme>/*.txt` listed there (path + short coverage blurb), grouped by theme. When you add or remove a scenario file, update this inventory in the same change.

### When the user asks to invent / suggest a scenario

1. Read `README.md` (behavior contract) and `E2E_SCENARIOS.md` (what already exists).
2. Pick a README-promised behavior that is **not** yet covered in `E2E_SCENARIOS.md`.
3. Implement it as `testdata/<theme>/<name>.txt` following the conventions below.
4. Add a row for it in `E2E_SCENARIOS.md` under the right theme.
5. **Do not run** the scenario (`task e2e`, `go test`, DinD, etc.). The user runs tests themselves.

### When the user asks to add a specific scenario

1. Check the request against `README.md` first.
2. If the requested behavior **disagrees** with the README (wrong effect, undocumented detail, contradicts soft-fail / events / cert rules, etc.): stop, explain the mismatch, and agree on the scenario in dialogue before writing files.
3. Once aligned: implement `testdata/<theme>/<name>.txt`, update `E2E_SCENARIOS.md`, and stop.
4. **Do not run** the scenario yourself.

### Do not run added scenarios

After adding or changing scenarios, do **not** execute e2e / `go test` / DinD to verify them unless the user explicitly asks. Just add the files and inventory entry.

## Writing a scenario

Each `testdata/<theme>/<name>.txt` is a testscript archive:

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
- In `stderr` / `stdout` patterns, avoid unescaped regex metacharacters (especially `(` `)` `.`) unless intentional — testscript matches as regexp.

## Commits

When the user asks to create a git commit, follow common open-source practices (Conventional Commits, as already used in this repo’s history):

- **Subject:** `<type>: <imperative summary>` — e.g. `test:`, `docs:`, `fix:`, `chore:`, `refactor:`.
- Keep the subject short (~72 chars); focus on **why** / intent, not a file list.
- Optional body: motivation and user-visible impact; wrap near 72 columns; blank line after the subject.
- **One logical change per commit** — do not mix unrelated edits.
- Never commit secrets, live credentials, or `bin/apinfra`.
- Commit **only** when the user asks; do not push unless asked.
- Prefer `git commit` with a HEREDOC for the message; do not skip hooks unless the user explicitly requests it.

## Agent behavior

- After harness Go changes: `go fmt ./...`. Do **not** run the e2e suite unless the user asks.
- Never commit secrets, real registry tokens, or live ACME credentials — fixtures use local Pebble EAB / stub directories only.
- Do not add Markdown docs the user did not ask for. Scenario inventory updates go only to `E2E_SCENARIOS.md` (never `README.md`).
