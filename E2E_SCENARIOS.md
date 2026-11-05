# E2E scenarios

Inventory of black-box scenarios in `testdata/*.txt` ([testscript](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript) archives against a local `file://` config Git repo and `bin/apinfra`).

This file is the **only** scenario inventory. Do not maintain a parallel list in `README.md`.

## Implemented (`testdata/`)

| File | Covers |
|------|--------|
| `happy_path.txt` | First deploy: compose up, proxy, `done duration=` |
| `idempotency.txt` | Second cycle with unchanged Git leaves stack running |
| `validation_rollback.txt` | Invalid `settings.yml` soft-fails; prior containers untouched |
| `port_stripping.txt` | Default: compose `ports` stripped (no public host publish) |
| `port_mapping.txt` | `allow-ports-mapping: true` keeps ports; unbound host IP → `127.0.0.1` |
| `nginx_conf_rollback.txt` | Failed ACME after conf rewrite restores prior nginx conf |
| `acme_cert_happy_path.txt` | Pebble/EAB issue in prod; PEMs under `certbot/conf/live/` |
| `acme_downgrade.txt` | Rate-limit → `.downgradeCertMode` prod→test; next cycle issues test |
| `acme_le_zerossl_fallback.txt` | LE 429 → ZeroSSL (Pebble) secondary keeps prod (no downgrade) |
| `cert_issue_failed.txt` | Non–rate-limit ACME fail → `.retries` block + `cert_issue_failed`; next cycle skips CA |
| `dns_preflight_skip.txt` | Empty `dig +short` skips ACME for the cycle (no retry / downgrade / `.installedCert`) |
| `heal_deleted_container.txt` | Two projects; `docker rm` one service in a 2-container stack → next cycle recreates it |
| `heal_stopped_container.txt` | Two projects; `docker stop` one service in a 2-container stack → next cycle starts it again |
| `project_disabled_downs.txt` | Two projects; `enabled: false` → `compose down` for that app; sibling stays running |
| `network_mode_host_stripped.txt` | `network_mode: host` stripped before start; runtime NetworkMode is not `host` |
