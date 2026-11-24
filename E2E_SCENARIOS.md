# E2E scenarios

Inventory of black-box scenarios in `testdata/<theme>/*.txt` ([testscript](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript) archives against a local `file://` config Git repo and `bin/apinfra`).

This file is the **only** scenario inventory. Do not maintain a parallel list in root `README.md` or `testdata/README.md`.

Themes are **one level** under `testdata/` (no nested theme folders). See `AGENTS.md` for naming and placement rules.

## Implemented

### `cycle/` — agent cycle

First deploy, idempotent re-run, validation soft-fail / leave prior state alone.

| File | Covers |
|------|--------|
| `happy_path.txt` | First deploy: compose up, proxy, `done duration=` |
| `idempotency.txt` | Second cycle with unchanged Git leaves stack running |
| `host_layout_perms.txt` | Full-cycle host path modes (apps/`0700`, compose/`.env` `0400`, nginx/ACME dirs `0755`/`0644` via glob for vhosts, secrets `0600`); legacy loose modes heal; curl inbound OK |
| `pending_markers_cleared.txt` | Healthy deploy + idempotent cycle: `infra/.apply_pending`, `apps/<name>/.apply_pending`, and `nginx/.reload_pending` absent; curl through proxy OK |
| `validation_rollback.txt` | Invalid `settings.yml` soft-fails; prior containers untouched |
| `target_url_validation.txt` | `proxy.target.url` + `inbound.port`, `url` XOR `service`, or URL with path → soft-fail; prior compose/nginx untouched |
| `proxies_validation.txt` | `proxy` XOR `proxies`, empty `proxies`, duplicate inbound domain/port → soft-fail; prior compose/nginx untouched |
| `stream_sni_validation.txt` | `stream.sni`: missing `default`, inbound `8443` reserved, route vs `domain-name`, duplicate `server-name` → soft-fail; prior inbound curl untouched |
| `git_login_failed.txt` | Config-repo clone HTTP 401 → `git_login_failed`; cycle aborted; prior Docker/Nginx untouched |

### `notifications/` — Outbound push (Shoutrrr)

| File | Covers |
|------|--------|
| `outbound_webhook.txt` | `notifications.*` + `generic+http` mock hook; `git_login_failed` delivered after cycle; second sticky-failure cycle delivers again |
| `send_fail_retries.txt` | Hook HTTP 500 → cycle still `done`; event kept; offset not advanced; next OK cycle delivers deferred `git_login_failed` |
| `filter_skip.txt` | `notifications.events=git_login_failed` skips `container_deployed` (empty hook); matching auth-fail is delivered |
| `no_historical_replay.txt` | Pre-seeded `~/.events` not replayed on first enable (EOF anchor); later new `git_login_failed` delivered |
| `hostpublicip_title.txt` | No `serverName` → alert title `hostPublicIP[namespace]` (`203.0.113.10[dev]`) |
| `events_all_deployed.txt` | Omitted `notifications.events` (default all) → first deploy pushes `container_deployed` |

### `docker/` — Docker / Compose apply

Ports, `network_mode`, volumes/socket policy, healing, enabling/disabling projects.

| File | Covers |
|------|--------|
| `port_stripping.txt` | Default: compose `ports` stripped (no public host publish) |
| `port_mapping.txt` | `allow-ports-mapping: true` keeps ports; unbound host IP → `127.0.0.1` |
| `port_mapping_reject_wildcard.txt` | `allow-ports-mapping: true` + `0.0.0.0` / `::` → soft-fail; prior containers untouched |
| `docker_login_failed.txt` | Full registry creds + failed `docker login` → `docker_login_failed`; compose written; no app up; inbound curl fails; prune skipped |
| `docker_login_skips_nginx.txt` | Proxy already up; add project + failed login → new inbound dead, prior curl OK; no "despite docker errors" |
| `apply_pending_retry.txt` | Compose change + failed login keeps same container on V1; fix host registry (no Git) → next cycle recreates container serving V2; fourth cycle stays on V2 |
| `proxy_apply_pending_retry.txt` | Proxy inbound.port change + failed login keeps same `apinfra-proxy` cid and old port; `infra/.apply_pending` stays; fix host registry (no Git) → next cycle restarts proxy on new port and clears pending |
| `network_mode_stripped.txt` | Any custom `network_mode` (`host`, `bridge`, …) stripped before start; written compose has none |
| `network_mtu_default.txt` | Omitted `proxyNetworkMTU` / `projectNetworkMTU` → real bridge MTU `1350` on `apinfra_proxy_net` and `<project>_net` |
| `network_mtu_override.txt` | `.apinfra` sets independent MTUs (`1400` / `1280`); `docker network inspect` matches; curl through proxy OK |
| `heal_deleted_container.txt` | Two projects; `docker rm` one service → next cycle recreates it + `container_deployed` (`added=worker`) |
| `heal_stopped_container.txt` | Two projects; `docker stop` one service → next cycle starts it again + `container_deployed` (`started=worker`) |
| `project_disabled_downs.txt` | Two projects; `enabled: false` → `compose down` for that app; sibling stays running |
| `worker_no_proxy.txt` | Project with no `proxies`/`proxy` still runs (no public ports); sibling with proxy answers curl |
| `target_url_keeps_compose.txt` | `proxy.target.url` + project compose folder → compose still applied; nginx uses sibling URL on proxy net (not edge Compose DNS); curl smoke OK |
| `container_name_target.txt` | `proxy.target.service.name` matches `container_name`; name stripped at runtime; nginx uses YAML service DNS |
| `service_name_target.txt` | `proxy.target.service.name` matches YAML compose service key; nginx DNS `<service_key>.apinfra_<project>`; curl OK |
| `no_new_privileges.txt` | `privileged: true` cleared; containers get `security_opt: no-new-privileges:true` |
| `include_extends_blocked.txt` | Compose `include` / `extends` to host paths ignored; leaked service never starts |
| `volume_policy_reject.txt` | Host bind + tmpfs → `security_violation` (`disallowed_bind_mount` / `disallowed_volume_type`); prior containers untouched |
| `named_volume_ok.txt` | Named Docker volume allowed; written compose keeps `type: volume`; `container_deployed`, no `security_violation` |
| `project_local_bind_ok.txt` | Relative `./file` + `./dir` binds (+ long-form `read_only: false` / short `:rw`) allowed; synced under `infra/apps/<project>/` as `0644`/`0755`; rewritten absolute + forced RO; named volume still OK; no `security_violation` |
| `project_local_bind_reject.txt` | `./` root, `./.env`, `./docker-compose.yml`, missing Git source → `disallowed_bind_mount`; prior container untouched |
| `project_local_bind_reject_traversal.txt` | `../` path traversal bind → `disallowed_bind_mount`; prior container untouched |
| `project_local_bind_reject_symlink.txt` | Symlink file as bind source → `disallowed_bind_mount`; prior container untouched |
| `project_local_bind_reject_symlink_dir.txt` | Dir bind containing symlink to in-project `.env` → `disallowed_bind_mount`; prior container untouched |
| `docker_sock_untrusted.txt` | Sock bind rejected when allowlist missing, tag-only listed, or digest not listed → `untrusted_docker_socket_mount`; prior containers untouched |
| `docker_sock_trusted.txt` | Digest in `~/.trusted_images` (case-insensitive, comments/blanks ok) + sock → deploy + `trusted_image_deployed` |
| `container_deployed.txt` | Successful app pull/up → `container_deployed` (`project`, `containers=api,web`, `added=api,web`; empty `removed`/`started` omitted); idempotent second cycle leaves `.events` unchanged |
| `container_deployed_diff.txt` | Compose change with stopped desired + new service + orphan → `container_deployed` (`containers=db,web`, `added=db`, `removed=worker`, `started=web`) |
| `container_deploy_failed.txt` | Compose change to missing image tag → `container_deploy_failed` (`project`, `containers`, `error`; empty diff fields omitted); `.apply_pending` kept; prior container still serves V1 |
| `container_removed.txt` | `enabled: false` → successful `compose down` + apps dir removed → `container_removed` (`project`, host-observed `containers`); sibling stays up |

### `nginx/` — Nginx config

Proxy conf write / restore / port TLS.

| File | Covers |
|------|--------|
| `nginx_conf_rollback.txt` | Same-domain `none` → failed prod ACME → live `:80` HTTP restored (no TLS left up) |
| `multi_proxies_routes.txt` | `projects[].proxies`: one stack, two port routes → curl both services |
| `proxies_per_route_cert.txt` | Project cert default `none` on port; route-level `mock` on domain → curl HTTP + HTTPS + 301 |
| `port_mock_https.txt` | `inbound.port` (9443) + `cert-issue-type: mock` → `mockcerts/port-9443/` + live HTTPS curl (plain HTTP fails) |
| `domain_http_none.txt` | Domain + `cert-issue-type: none` → `.certs.json` `none`; live HTTP curl, no 301 / no HTTPS |
| `domain_prod_switch_fail_keeps_proxy.txt` | Prod domain A live (curl); add domain B + non-429 ACME fail → block B, curl A still OK / B not |
| `proxy_stops_after_project_removed.txt` | Remove project from settings → down + conf gone; curl old port fails, sibling port still OK |
| `proxy_stops_after_project_disabled.txt` | `enabled: false` → down + conf gone; curl old port fails, sibling port still OK |
| `reload_pending_retry.txt` | Conf+`-t` OK but `nginx -s reload` fails (docker stub once) → curl inbound ≠ V2; unchanged next cycle retries reload → curl V2; fourth cycle stays V2 (REL-02) |
| `target_url_skip_compose.txt` | URL-only (no folder) → SkipCompose; HTTP conf/`Host` from URL to sibling on proxy net; curl smoke OK |
| `target_url_https_upstream.txt` | URL-only `https://` sibling upstream + mock → `:80` 301, `proxy_ssl_server_name`/`Host`; live TLS + redirect smoke |
| `stream_sni_passthrough.txt` | `stream.sni.enabled` + route + `default: reject`: managed mock domain HTTPS + `:80` 301; distinct TLS body/cert on mqtt SNI; unmapped SNI fails; compose-only broker on `apinfra_proxy_net`, no public ports; host does not publish `8443` |
| `stream_sni_compose_default.txt` | Compose `stream.sni.default` (no routes): unmapped mqtt SNI reaches TLS backend; managed shop HTTPS + `:80` 301 still OK |

### `certs/` — Certificates / ACME

Issue, fallback, downgrade, retries, DNS preflight, mock self-signed.

| File | Covers |
|------|--------|
| `acme_cert_happy_path.txt` | Pebble/EAB issue in prod; PEMs under `certbot/conf/live/`; `.certs.json` has `active_mode` + `certs.prod` PEM cache + accounts |
| `acme_downgrade.txt` | Rate-limit → `cert_rate_limited` (`domain`, `from=prod`, `to=test`) + `.downgradeCertMode` prod→test; next cycle issues test |
| `acme_le_zerossl_fallback.txt` | LE 429 → ZeroSSL (Pebble) secondary keeps prod (no downgrade) |
| `cert_issue_failed.txt` | Non–rate-limit ACME fail → `.retries` block + `cert_issue_failed`; next cycle skips CA |
| `dns_preflight_skip.txt` | Empty `dig +short` skips ACME for the cycle (no retry / downgrade / `.certs.json` / cert events; `container_deployed` ok) |
| `dns_preflight_public_ip.txt` | Configured `hostPublicIP` matches or mismatches dig IP -> pass or skip cycle (mismatch: no cert events) |
| `mock_domain_cert.txt` | Domain + `mock` → local PEMs under `mockcerts/<domain>/` (no ACME); live HTTPS + `:80` 301 |
| `prod_mode_switch_retains.txt` | Intentional `prod → test → prod`: staging on disk mid-step; prod restored from `.certs.json` without second obtain |
| `certs_json_cold_restore.txt` | Delete `live/` PEMs; next cycle materializes from `.certs.json` without ACME obtain |
| `prod_soft_ttl_reissue.txt` | Backdate `issued_at` past soft TTL (live PEMs + prod cache kept); `proxyReadyAfter=1s` → second ACME obtain (not restore) |

### `env/` — Host / project environment

Host `.env` project-prefix injection (`TO_UPPER(name)__`) and sidecar `.env` key normalization.

| File | Covers |
|------|--------|
| `host_overrides_sidecar.txt` | `STOREFRONT__MARKER` on host overrides sidecar `MARKER` in `infra/apps/storefront/.env` |
| `host_nested_prefix_isolation.txt` | `app` / `app_billing`: `APP__TOKEN` and `APP_BILLING__TOKEN` stay project-scoped (no `BILLING_TOKEN` leak into `app`) |
| `sidecar_dot_normalize.txt` | Sidecar `web.image.tag` → `WEB_IMAGE_TAG` in applied project `.env` |

### `security/` — Credential hygiene in logs and host files

Redaction tests only: sentinels must not appear in stderr or agent-written `$HOME` files (input `.apinfra*` excluded). All scenarios set `useExecLogs=true` so credential checks cover `[exec]` journaling too. Local `git-authfail` (HTTP 401) is only a fast soft-fail trigger — these scenarios do **not** assert a successful clone or real GitHub auth.

| File | Covers |
|------|--------|
| `userinfo_password.txt` | Wrong `user:password@host` (+ `repoToken`) + `useExecLogs` → `http://***@host/...` |
| `userinfo_token_only.txt` | Wrong `token@host` + `useExecLogs` → `http://***@host/...` |
| `userinfo_token_as_password.txt` | Wrong `x-access-token:token@host` + `useExecLogs` → `http://***@host/...` |
| `userinfo_token_no_at.txt` | Malformed `http://ghp_…/host/repo` (no `@`) + `useExecLogs` → `http://***` + bare PAT scrubbed |
| `userinfo_at_in_password.txt` | Wrong password containing `@` + `useExecLogs` → `http://***@host/...` (last `@`) |
| `execlogs_git_bad_creds.txt` | Wrong git user/password + `useExecLogs`; sentinel absent (clone is go-git, not shell exec) |
| `registry_token.txt` | Wrong registry token + `useExecLogs`; `[exec] … docker login … --password-stdin` without sentinel |
| `execlogs_registry_bad_creds.txt` | Wrong registry user/password + `useExecLogs`; asserts `[exec]` / `FAIL` cmd lines, no password in logs |
