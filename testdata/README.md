# apinfra

**apinfra** is a lightweight GitOps agent that deploys **docker-compose** apps on a **single host**, with automatic **Nginx** reverse-proxy routing and **Let's Encrypt** / **ZeroSSL** certificates.

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

## Why apinfra? (The Webhookless GitOps Approach)

* **Lightweight:** Unlike heavy-weight self-hosted PaaS solutions, **apinfra** needs no web UI on the server and adds no web-server overhead — so practically all RAM and CPU stay available for your applications.
* **Pull model:** The agent fetches configuration from Git itself, so you do not open inbound management ports, webhooks, or panels — day-to-day management goes through the configuration Git repository; no server login is required for ongoing operations.
* **Single source of truth in Git:** All settings, routes, and application configs live in one Git repository. That lets you bring up any server in the desired configuration in minutes and gives you the full GitOps toolkit — version history, reviews, rollbacks, and reproducible deploys.


## Key Features

* **GitOps Pull Workflow:** Desired server state lives in the configuration Git repository (`settings.yml` + compose files); the host periodically **pulls** and applies it.
* **Multiple Compose Files:** Ability to deploy several independent apps from one config repository — each with its own `docker-compose.yml` (and optional `.env`), plus shared `settings.yml`, with Nginx routing and Let's Encrypt / ZeroSSL TLS on the host.
* **Automatic SSL:** Let's Encrypt / ZeroSSL certificate management.
* **Nginx Reverse Proxy:** Routes public domains (HTTPS/HTTP) or host ports (HTTP, or HTTPS with mock self-signed certs) to containers by Docker DNS name and target port.
* **Port / host-network exposure guard:** By default, all `ports` mappings and any `network_mode: host` in docker-compose are stripped, so containers are not accidentally published on the host network stack; public access goes only through the managed Nginx routes. When needed, you can still configure port forwarding bound to a specific IP (`allow-ports-mapping: true`).
* **Lightweight & Fast:** Built with Go — minimal overhead, low memory footprint, and fast execution.

## How It Works

![How apinfra works](docs/how-it-works.svg)

**Flow in short:**

1. You declare projects in `settings.yml` and place each app's `docker-compose.yml` (and optional `.env`) in a matching folder.
2. The agent on the host **pulls** (clones) the configuration repository and selects the environment folder from `namespace`.
3. It validates `settings.yml` and all `docker-compose.yml` files; on any failure the cycle stops with no changes.
4. If validation succeeds, the agent compares the server state with `settings.yml` and the `docker-compose` files from the config repository. When they already match, it changes nothing; otherwise it brings up or removes containers, reconfigures the proxy, and applies related updates as needed — e.g. `domain-name` → HTTPS (or plain HTTP when `cert-issue-type: none`), or `inbound.port` → HTTP / mock HTTPS on that host port.
5. The server is ready to accept traffic and forward requests to the corresponding containers of your applications.
6. All of these steps repeat every minute via a systemd timer.

## Get Started

### 1. System Requirements

To deploy `apinfra`, you need a **clean Linux virtual machine or bare-metal server** that meets the following criteria:

* **OS:** Ubuntu 22.04 LTS (or newer).
* **Hardware:**
    * 1 vCPU and 512 MB RAM minimum.
    * *Note: A 1 GB+ Swap file is strictly required if running on 512 MB RAM.*
* **Network:** Persistent outbound internet access (required to fetch Git repositories and Docker images).

### 2. Installation

### Option 1: Fast Installation via GitHub Releases (Recommended)

Connect to your clean Linux server (Ubuntu/Debian) via SSH and execute the following commands:

```bash
# 1. Download the latest package version
curl -L -O https://github.com

# 2. Update package indexes and install
sudo apt update
sudo apt install -y ./apinfra_*.deb
```

---

### Option 2: Local Build and Manual Installation

#### Building the Package

```bash
# Run from the apinfra project folder on your local computer
./build/deb/package.sh

# The script will generate a package in the dist directory:
# → dist/apinfra_<version>_amd64.deb
```

#### Transfer and Install on the Server
Copy the generated file to your target server (e.g., using `scp`) and install it:

```bash
# On your local machine:
scp dist/apinfra_*.deb user@your-server-ip:/tmp/

# On the remote server:
sudo apt update
sudo apt install -y /tmp/apinfra_*.deb
```

### After Installation

The package sets up the following on the server:
1. Creates a system user **`ap`**.
2. Deploys a **default example application** so you can verify the install right away.
3. Installs a **systemd** user service and timer for `ap` that runs the agent **once per minute**.
4. Provisions the home directory layout:
    * `/home/ap/apinfra` — Agent binary.
    * `/home/ap/.apinfra` — Host agent settings (repo, registry, …).
    * `/home/ap/.env` — Host application environment variables (secrets / project-prefixed keys).
    * `/home/ap/infra/` — Runtime directory for Nginx, Compose files, and certificates.

### Verifying the Installation

By default, immediately after installation, the agent safely provisions an initial demo layout to ensure your environment is working. It points to our official public [apinfra-demo-conf repository](https://github.com/uservseatb/apinfra-demo-conf) and deploys a lightweight [apinfra-demo](https://github.com/uservseatb/apinfra-demo) page.

Wait about 10–15 seconds for the containers to spin up, then open your browser at:

```text
http://<YOUR_SERVER_IP>:3000
```

If the default **apinfra-demo** page loads successfully, it means the agent, Nginx reverse proxy, and the demo container are fully operational.

Once verified, you can instantly replace the demo path with your own Git repository by adjusting the `repo` string inside `/home/ap/.apinfra`.

## Connecting Your Own Project (Server Configuration)

To connect your own projects, create a **configuration Git repository**, then point the server at it — as described below.

### Configuring `/home/ap/.apinfra`

#### 1. Configuration Repository Settings (Git)

By default, the agent points to a public example repository. To deploy your own applications, change these keys:

<table>
<thead>
<tr>
<th align="left">Property</th>
<th align="left">Description</th>
</tr>
</thead>
<tbody>
<tr bgcolor="#f6f8fa">
<td><code>namespace</code></td>
<td>The name of the environment folder inside your configuration repository (e.g., <code>dev</code>, <code>prod</code>). You can leave this as default (<code>dev</code>) during initial setup.</td>
</tr>
<tr>
<td><code>repo</code></td>
<td>The HTTPS URL of your <strong>custom configuration</strong> Git repository.</td>
</tr>
<tr bgcolor="#f6f8fa">
<td><code>repoToken</code></td>
<td>Optional. Provide a Personal Access Token (PAT) only if the configuration repository is <strong>private</strong>. Leave empty for public HTTPS repos — the agent will not attempt git authentication.</td>
</tr>
<tr>
<td><code>repoBranch</code></td>
<td>Optional. Git branch to clone. Default: <code>main</code>. If the branch does not exist in the repository, the cycle stops with an error.</td>
</tr>
</tbody>
</table>

The agent clones your configuration Git repository into memory and processes the environment directory corresponding to the host's `namespace` variable.

**Example repository directory structure:**

```text
dev/
  settings.yml
  storefront/
    docker-compose.yml
    .env
  dashboard/
    docker-compose.yml
    .env
  secure-api/
    docker-compose.yml
    .env
  worker/
    docker-compose.yml
    .env
prod/
  settings.yml
  ...
```

> [!NOTE]
> Environment folders (`dev`, `prod`, etc.) must sit directly at the **root** of the configuration repository.

Each project folder (e.g. `storefront/`, `dashboard/`, `secure-api/`, `worker/`) is **one project** and must match a `projects[].name` entry in `settings.yml`. Inside the folder: `docker-compose.yml` and an optional `.env`. The `proxy.target` block in `settings.yml` points at a service in that compose file — the service that listens on the declared container port.

##### `settings.yml` Example

`settings.yml` declares projects and wires them to inbound routing:

```yaml
version: "1.0"
projects:
  - name: storefront
    enabled: true
    proxy:
      inbound:
        domain-name: storefront.example.com  # Domain mode → HTTPS
      target:
        service-name: web                    # matches storefront/docker-compose.yml
        port: 80                             # container port the web service listens on
    config:
      cert-issue-type: prod                  # domain: mock | prod | test | none (default: mock)
      # cert-update-time-sec: 90             # optional override (seconds)
      cert-acme-email: ops@example.com       # required for prod/test + domain-name

  - name: dashboard
    enabled: true
    proxy:
      inbound:
        port: 3333                           # Host port Nginx listens on
      target:
        service-name: app                    # matches dashboard/docker-compose.yml
        port: 3000                           # container port the app service listens on
    config:
      cert-issue-type: none                  # port: mock | none only (default: none = plain HTTP)

  - name: secure-api
    enabled: true
    proxy:
      inbound:
        port: 8443                           # Host port with mock self-signed TLS
      target:
        service-name: api
        port: 8080
    config:
      cert-issue-type: mock                  # self-signed HTTPS on :8443 (browsers will warn)

  - name: worker
    enabled: true                            # No proxy — compose runs without Nginx routing
    config: {}
```

In this example:

* **storefront** — available at `https://storefront.example.com` (Let's Encrypt `prod` cert via `cert-acme-email`; Nginx terminates TLS and proxies to the `web` container on port `80`).
* **dashboard** — available at `http://<server-ip>:3333` (plain HTTP; `cert-issue-type: none`).
* **secure-api** — available at `https://<server-ip>:8443` (mock self-signed certificate; port routing allows only `mock` or `none`).
* **worker** — not exposed from outside (no `proxy`). It still runs and can reach the internet — e.g. a pull-model bot, cron-style job, or background worker.

Full field reference: [settings.yml Field Schema](#settingsyml-field-schema).

**Example `storefront/docker-compose.yml`** (service `web` listens on port `80`):
```yaml
services:
  web:
    image: ghcr.io/example/storefront-web:${WEB_IMAGE_TAG}
    expose:
      - "80"
    restart: unless-stopped
```

**Example `dashboard/docker-compose.yml`** (service `app` listens on port `3000`):
```yaml
services:
  app:
    image: ghcr.io/example/dashboard:${APP_IMAGE_TAG}
    expose:
      - "3000"
    restart: unless-stopped
```

**Example `secure-api/docker-compose.yml`** (service `api` listens on port `8080`):
```yaml
services:
  api:
    image: ghcr.io/example/secure-api:${API_IMAGE_TAG}
    expose:
      - "8080"
    restart: unless-stopped
```

**Example `worker/docker-compose.yml`** (no HTTP entrypoint — e.g. a background job or bot):
```yaml
services:
  worker:
    image: ghcr.io/example/worker:${WORKER_IMAGE_TAG}
    restart: unless-stopped
```

> [!NOTE]
> Before starting a stack, the agent always strips `container_name` from every service so host uniqueness comes from the Compose project name (`apinfra_<project>`). You may still set `container_name` in the config-repo compose file so `proxy.target.service-name` can match it (or the YAML service key); runtime DNS on `apinfra_proxy_net` is `<service_key>.apinfra_<project>`.

#### 2. Private Container Registry Settings (Docker)

<table>
<thead>
<tr>
<th align="left">Property</th>
<th align="left">Description</th>
</tr>
</thead>
<tbody>
<tr bgcolor="#f6f8fa">
<td><code>containerRegistryHost</code></td>
<td>Optional. Full URL of your private image registry (e.g., <code>https://ghcr.io</code>).</td>
</tr>
<tr>
<td><code>containerRegistryUser</code></td>
<td>Optional. Registry authentication username.</td>
</tr>
<tr bgcolor="#f6f8fa">
<td><code>containerRegistryToken</code></td>
<td>Optional. Registry personal access token or password.</td>
</tr>
</tbody>
</table>

> [!NOTE]
> Currently, `apinfra` supports authentication for **only one** private Docker registry. If your compose files use public images exclusively, leave these three registry fields empty — the agent skips `docker login` and continues without failing.

---

### Configuring `/home/ap/.env`

You can inject host-specific environment variables or production secrets directly into your projects' containers from this file.

* **Crucial Rule:** Any environment variable intended for a specific application inside your compose files **must be prefixed with the project's name** (converted to uppercase) followed by an underscore. The prefix is stripped before compose interpolation, so the project's `docker-compose.yml` and its `.env` in the config repository use the **unprefixed** name.
* *Example:* If your project is named `storefront` in `settings.yml`, and you want to pass a database host variable, you must declare it in this file as `STOREFRONT_DB_HOST=your-host-value`.
* If the same variable is also defined in the project's `.env` inside the configuration repository, the value from `/home/ap/.env` on the host **overrides** it.

```yaml
# config repo: /dev/storefront/docker-compose.yml
services:
  web:
    image: ghcr.io/example/storefront-web:${IMAGE_TAG}
    environment:
      - DB_PASSWORD=${DB_PASSWORD}
    # ...
```

```bash
# config repo: /dev/storefront/.env
IMAGE_TAG=0.3.7
# ...
```

```bash
# server host: /home/ap/.env
STOREFRONT_DB_PASSWORD=supersecret
# ...
```


### Applying Changes

After modifying `/home/ap/.apinfra` or `/home/ap/.env`, you can either wait for the system timer to trigger the next deployment cycle automatically or restart the service manually to apply your changes immediately (as user `ap`):

```bash
systemctl --user start --no-block apinfra.service
```

### Viewing Service Logs

To inspect or follow the `apinfra` service logs (as user `ap`):

```bash
journalctl --user -u apinfra.service -f
```

Drop `-f` to print recent logs without following, or add `-n 200` to limit the output.

## Architecture & Design

### Security Design & Trade-offs (Pragmatic Isolation)

**apinfra** focuses on a pragmatic approach to security, balancing strong host-level protection with high operational stability for startups and MVP environments.

* **Layered Network Isolation:** Projects are strictly separated into independent `<project>_net` bridge networks. Databases and internal services stay reachable only inside Docker networks — they are not published on the host.
* **Single Shared Proxy Net:** To keep the Nginx proxy up and avoid restarting it on every deploy cycle, all public-facing services share one `apinfra_proxy_net`. This assumes all apps on the host belong to the same trusted tenant.
* **Zero Web UI Attack Surface:** No web admin panel or open management API on the internet — the agent only pulls configuration locally.
* **Strict Port Mapping:** By default, any explicit `ports:` mappings in the application's `docker-compose.yml` are automatically stripped before the stack is started, ensuring no accidental database or service exposure via the host network stack. If a project genuinely requires host port publishing, set `config.allow-ports-mapping: true` in `settings.yml`. Then mappings without a host IP are bound to `127.0.0.1`; mappings that already include a host IP are kept as written — except unspecified addresses `0.0.0.0` / `::`, which are rejected.
* **Host-Network Control:** Any `network_mode: host` in the application's `docker-compose.yml` is always stripped before the stack is started — there is no opt-in — so stacks stay on bridge networks (`<project>_net` / `apinfra_proxy_net`).
* **Container Name Control:** Any `container_name` in the application's `docker-compose.yml` is always stripped before the stack is started. Fixed names collide across projects; uniqueness comes from Compose project `-p apinfra_<name>`. `proxy.target.service-name` may still match the original `container_name` or the YAML service key when resolving the proxy target (matching runs before the strip).
* **Privilege Escalation Protection:** Every container is started with `no-new-privileges: true`. This prevents processes inside the container from gaining new privileges (e.g., via setuid/setgid binaries) after startup. Even if a container runs as root internally to initialize services or bind ports, this layer protects the host system by blocking runtime privilege escalation exploits and container escape vectors.

### Project `.env` and Variable Normalization
Each project folder may include an optional `.env` next to its `docker-compose.yml`. In that file you can use a dot-separated "Java-style" notation. Before evaluating the variables, the agent **normalizes** them: it replaces dots with underscores and converts strings to uppercase.

Example `.env` key in your configuration repository (`dev/storefront/.env`):
```env
web.image.tag=dev-v0.0.203
```

Your `docker-compose.yml` must reference the normalized key format:
```yaml
image: ghcr.io/example/storefront-web:${WEB_IMAGE_TAG}
```

### Certificate issuance

State files under `/home/ap/`:

| File | Role |
|------|------|
| `/home/ap/.installedCert` | Last **successful** issue (domain → time\|mode). Updated only after a successful issuance. |
| `/home/ap/.downgradeCertMode` | After ACME **rate limit** or failure of the other CA — temporary target mode (`prod`→`test`→`mock`→`none`) with an `until` timestamp. While active, overrides `cert-issue-type` from `settings.yml`. Expires automatically. Can also be useful while DNS is still being configured / propagating. |
| `/home/ap/.retries` | Attempt counter for errors (not rate limit), plus `blocked` and `blocked_until`. |

Retry / block knobs in `.apinfra` (positive integers; defaults shown):

| Property | Default | Description |
|----------|---------|-------------|
| `certs.triesCount` | `3` | Non–rate-limit failures before `blocked` (one attempt per agent cycle) |
| `certs.blockHours` | `24` | Hours to keep `blocked` after `triesCount` |
| `certs.rateLimitBlockHours` | `24` | Hours for mode downgrade after ACME rate limit / HTTP 429 |

**DNS preflight** (hostname `prod`/`test` only): Before issuing a certificate via ACME, apinfra checks that the domain resolves on the host (`dig +short <domain>`). Empty answer → skip this cycle. If `certs.publicIP` is set, that IP must appear among dig's results; if unset, any resolved IP is enough. On failure, issuance is skipped for this cycle (no retry bump, no downgrade).

**On any issuance failure for a domain:** the cycle does not abort. The previous nginx `conf.d` file for that service is restored from the pre-cycle snapshot when one existed; `.installedCert` for that domain is left unchanged. If there was no prior conf (first issue), nothing is restored and `.installedCert` is not created.

**ACME rate limit** (`urn:ietf:params:acme:error:rateLimited` / HTTP 429):

1. If ZeroSSL EAB credentials are also set in `.apinfra`, apinfra tries the **other** CA at the same mode before downgrading.
2. If both providers fail at that mode (or only one CA is configured), event `cert_rate_limited` is written (`domain`, `from`, `to`).
3. `.downgradeCertMode` is set to the next lower mode for `certs.rateLimitBlockHours` (`prod` → `test` → `mock` → `none`).
4. The next timer run tries the downgraded mode (again with both providers when configured). `.installedCert` is updated only after a successful issue in that mode (`mock` is local self-signed and does not need ACME, so it usually succeeds). If issuance is still rate-limited at `mock`, the next step is temporary `none` (plain HTTP, no TLS) until the downgrade TTL expires.
5. After `certs.rateLimitBlockHours` elapses, the downgrade mode expires and the mode from `settings.yml` is used again.

**Any other issuance error:**

1. `attempts` in `/home/ap/.retries` is incremented (one attempt per agent run).
2. After `certs.triesCount` failures: `blocked: true`, `blocked_until` = now+`certs.blockHours`, and event `cert_issue_failed`.
3. While blocked, apinfra **skips** further issue attempts for that domain.
4. After `blocked_until`, the block clears automatically; you can also delete the domain entry under `certs:` in `.retries`.


## settings.yml Field Schema

### Root Level

<table>
<thead>
<tr>
<th align="left">Property</th>
<th align="left">Description</th>
</tr>
</thead>
<tbody>
<tr bgcolor="#f6f8fa">
<td><code>version</code></td>
<td><strong>Required.</strong> Schema version string. Currently used: <code>"1.0"</code>.</td>
</tr>
<tr>
<td><code>projects</code></td>
<td><strong>Required.</strong> Array of projects to be deployed.</td>
</tr>
</tbody>
</table>

### Project Level

<table>
<thead>
<tr>
<th align="left">Property</th>
<th align="left">Description</th>
</tr>
</thead>
<tbody>
<tr bgcolor="#f6f8fa">
<td><code>name</code></td>
<td><strong>Required.</strong> Unique project identifier (case-insensitive). Used as a prefix for environment variables (<code>TO_UPPER(name) + "_"</code>).</td>
</tr>
<tr>
<td><code>enabled</code></td>
<td>Optional. Defaults to <code>true</code>. If <code>false</code>, the project is bypassed during deployment cycles.</td>
</tr>
<tr bgcolor="#f6f8fa">
<td><code>proxy</code></td>
<td>Optional. Routing and reverse proxy rules. If omitted, no Nginx proxy configs will be generated.</td>
</tr>
<tr>
<td><code>config</code></td>
<td>Optional. Per-project settings (certificate issuance, port publishing, etc.).</td>
</tr>
</tbody>
</table>

### `proxy` Block Rules

`proxy.inbound.domain-name` and `proxy.inbound.port` are **mutually exclusive (XOR)** — exactly one must be defined.

<table>
<thead>
<tr>
<th align="left">Property</th>
<th align="left">Description</th>
</tr>
</thead>
<tbody>
<tr bgcolor="#f6f8fa">
<td><code>proxy.inbound.domain-name</code></td>
<td>Public hostname used for Nginx routing and HTTPS certificate tracking.</td>
</tr>
<tr>
<td><code>proxy.inbound.port</code></td>
<td>Distinct host port that Nginx listens to for this specific app. HTTP by default (<code>cert-issue-type: none</code>); set <code>mock</code> for self-signed HTTPS on that port.</td>
</tr>
<tr bgcolor="#f6f8fa">
<td><code>proxy.target.service-name</code></td>
<td><strong>Required</strong> when <code>proxy</code> is set. Matched against the YAML service key or <code>container_name</code> in the config-repo compose file (before the agent strips <code>container_name</code>). Nginx reaches the target as <code>&lt;service_key&gt;.apinfra_&lt;project&gt;</code> on <code>apinfra_proxy_net</code>.</td>
</tr>
<tr>
<td><code>proxy.target.port</code></td>
<td><strong>Required</strong> when <code>proxy</code> is set. Internal container target port.</td>
</tr>
</tbody>
</table>

### `config` Settings (per project)

Per-project settings under `projects[].config`. Certificate fields apply **per project**, not globally on the host.

Allowed `cert-issue-type` values depend on the inbound routing mode:

| Inbound | Allowed `cert-issue-type` | Default when omitted |
| --- | --- | --- |
| `proxy.inbound.domain-name` | `mock`, `prod`, `test`, `none` | `mock` |
| `proxy.inbound.port` | `mock`, `none` only (`prod` / `test` are rejected) | `none` |

> [!NOTE]
> Port routing cannot use Let's Encrypt (`prod` / `test`) — there is no public hostname for ACME. Use `mock` for self-signed HTTPS on the host port, or `none` for plain HTTP.

<table>
<thead>
<tr>
<th align="left">Property</th>
<th align="left">Description</th>
</tr>
</thead>
<tbody>
<tr bgcolor="#f6f8fa">
<td nowrap><code>cert-issue-type</code></td>
<td>Certificate mode: <code>mock</code>, <code>prod</code>, <code>test</code>, or <code>none</code>. See the table above for which values apply to domain vs port routing.</td>
</tr>
<tr>
<td nowrap><code>cert-update-time-sec</code></td>
<td>Optional. Override refresh interval in seconds. If omitted, uses the default for <code>cert-issue-type</code> (see table below). Ignored when <code>none</code>.</td>
</tr>
<tr bgcolor="#f6f8fa">
<td nowrap><code>cert-acme-email</code></td>
<td>ACME / Let's Encrypt registration email. Required when <code>cert-issue-type</code> is <code>prod</code> or <code>test</code> and <code>domain-name</code> is set; optional for <code>mock</code> / <code>none</code>.</td>
</tr>
<tr>
<td nowrap><code>allow-ports-mapping</code></td>
<td>Optional. Defaults to <code>false</code>. When <code>false</code> (default) or omitted, any explicit <code>ports:</code> mappings in the project's <code>docker-compose.yml</code> are stripped before the stack is started. Set to <code>true</code> only if the project must publish host ports: ports without a host IP bind to <code>127.0.0.1</code>; existing host IPs are kept except <code>0.0.0.0</code> / <code>::</code>, which are rejected.</td>
</tr>
</tbody>
</table>

`cert-issue-type` values and default refresh intervals:

<table>
<thead>
<tr>
<th align="left">Value</th>
<th align="left">Default <code>cert-update-time-sec</code></th>
<th align="left">Description</th>
</tr>
</thead>
<tbody>
<tr bgcolor="#f6f8fa">
<td><code>mock</code></td>
<td><code>2592000</code> (1 month)</td>
<td>Self-signed certificates under <code>infra/mockcerts/&lt;name&gt;/</code> (no ACME). For domain routing, <code>&lt;name&gt;</code> is the hostname; for port routing, <code>port-&lt;inbound.port&gt;</code>. Default for domain mode. Browsers will warn; use for local/dev HTTPS.</td>
</tr>
<tr>
<td><code>prod</code></td>
<td><code>5184000</code> (2 months)</td>
<td>Real production Let's Encrypt certificates. Domain routing only.</td>
</tr>
<tr bgcolor="#f6f8fa">
<td><code>test</code></td>
<td><code>2592000</code> (30 days)</td>
<td>Let's Encrypt staging certificates (ACME staging directory via lego). Domain routing only. Use to validate ACME without rate limits; re-issue replaces an existing production certificate with a staging one.</td>
</tr>
<tr>
<td><code>none</code></td>
<td>—</td>
<td>Do not use TLS. Nginx serves plain HTTP (no HTTPS / no redirect). <code>.installedCert</code> is updated to <code>none</code> (previous issue date kept). Default for port mode.</td>
</tr>
</tbody>
</table>

## Events

Operational events are appended as JSON Lines to `~/.events` (for user `ap`: `/home/ap/.events`). Each line is a JSON object with `ts` (UTC RFC3339), `kind`, and event-specific string fields. The agent never fails the systemd timer unit for these soft errors; each case aborts or degrades only the relevant part of the cycle.

| Event | When | Fields | Effect |
|-------|------|--------|--------|
| `git_login_failed` | Config-repo clone fails due to authentication/authorization (not network, missing branch, or timeout) | `repo` — clone URL | The **entire update cycle is aborted** — Docker and Nginx on the host are left untouched. Other clone failures also abort the cycle but do **not** write an event. |
| `docker_login_failed` | `containerRegistryHost` / `containerRegistryUser` / `containerRegistryToken` are all set in `.apinfra`, a compose pull/up is needed this cycle, and `docker login` fails | `registry` — registry host | Compose files for this cycle may already be written; pull/up/down stop; **nginx and image prune are skipped**. Incomplete registry credentials skip login (no event, cycle continues). |
| `cert_rate_limited` | ACME rate limit / HTTP 429 after both CAs fail at the current mode (or only one CA is configured) | `domain`, `from`, `to` — mode before/after downgrade | Sets `.downgradeCertMode` to the next lower mode for `certs.rateLimitBlockHours` (`prod`→`test`→`mock`→`none`). When the override is already `none`, Nginx serves plain HTTP and skips ACME for that domain until the TTL expires; other domains continue. |
| `cert_issue_failed` | After `certs.triesCount` non–rate-limit issuance failures for a domain (one attempt per agent cycle) | `domain`, `attempts`, `mode`, `blocked_until` | Writes `blocked: true` in `.retries` until `blocked_until` (now+`certs.blockHours`); further issue attempts for that domain are skipped while blocked. Other domains continue. |

## Additional Settings

Optional keys in `/home/ap/.apinfra` for logging verbosity, ACME CA selection, and operation timeouts.

<table>
<thead>
<tr>
<th align="left">Property</th>
<th align="left">Default</th>
<th align="left">Description</th>
</tr>
</thead>
<tbody>
<tr bgcolor="#f6f8fa">
<td><code>useExecLogs</code></td>
<td><code>false</code></td>
<td>Optional. When <code>true</code>, detailed stdout and stderr of command invocations are written to the system journal (<code>journalctl</code>).</td>
</tr>
</tbody>
</table>

### Certificates (ACME CA)

Per-project certificate **mode** (`cert-issue-type`, email, refresh interval) lives in `settings.yml`. The host `.apinfra` selects which ACME certificate authority issues `prod` / `test` certificates, optional DNS preflight IP, and optional directory URL overrides. Retry / downgrade knobs (`certs.triesCount`, `certs.blockHours`, `certs.rateLimitBlockHours`) are documented under [Certificate issuance](#certificate-issuance).

<table>
<thead>
<tr>
<th align="left">Property</th>
<th align="left">Default</th>
<th align="left">Description</th>
</tr>
</thead>
<tbody>
<tr bgcolor="#f6f8fa">
<td><code>certs.provider</code></td>
<td><code>letsencrypt</code></td>
<td>Primary ACME CA: <code>letsencrypt</code> or <code>zerossl</code>. ZeroSSL as primary requires EAB credentials below.</td>
</tr>
<tr>
<td><code>certs.zerossl.AcmeKid</code></td>
<td>—</td>
<td>ZeroSSL External Account Binding (EAB) key id. Required when <code>certs.provider=zerossl</code>. Obtain from the ZeroSSL panel. If set while the primary is Let's Encrypt, ZeroSSL is used as the rate-limit fallback CA before mode downgrade.</td>
</tr>
<tr bgcolor="#f6f8fa">
<td><code>certs.zerossl.AcmeHmac</code></td>
<td>—</td>
<td>ZeroSSL EAB HMAC key. Required together with <code>AcmeKid</code> when using ZeroSSL (primary or fallback).</td>
</tr>
<tr>
<td><code>certs.publicIP</code></td>
<td>—</td>
<td>Optional. If set, DNS preflight (<code>dig +short</code>) before ACME requires this IP among the answers. If unset, any resolved IP is enough. See [Certificate issuance](#certificate-issuance).</td>
</tr>
<tr bgcolor="#f6f8fa">
<td><code>certs.letsencrypt.directory</code></td>
<td><code>https://acme-v02.api.letsencrypt.org/directory</code></td>
<td>Optional. Let's Encrypt production ACME directory URL (lego <code>CADirURL</code>).</td>
</tr>
<tr>
<td><code>certs.letsencrypt.stagingDirectory</code></td>
<td><code>https://acme-staging-v02.api.letsencrypt.org/directory</code></td>
<td>Optional. Let's Encrypt staging ACME directory URL used for <code>cert-issue-type: test</code>.</td>
</tr>
<tr bgcolor="#f6f8fa">
<td><code>certs.zerossl.directory</code></td>
<td><code>https://acme.zerossl.com/v2/DV90</code></td>
<td>Optional. ZeroSSL ACME directory URL.</td>
</tr>
</tbody>
</table>

### Timeouts

Omit a key or leave it empty for the default. Values are Go durations (`30s`, `1m`, `10m`) and must be greater than zero — no upper bound. Invalid values (zero, negative, or unparseable) soft-fail the cycle: a warning is logged, nothing is applied, and the next timer tick retries. Every CLI/`Exec` path uses one of these timeouts.

<table>
<thead>
<tr>
<th align="left">Property</th>
<th align="left">Default</th>
<th align="left">Description</th>
</tr>
</thead>
<tbody>
<tr bgcolor="#f6f8fa">
<td><code>gitTimeout</code></td>
<td><code>1m</code></td>
<td>Config-repo clone. On timeout the cycle soft-fails (no event; only auth failures write <code>git_login_failed</code>).</td>
</tr>
<tr>
<td><code>dockerLoginTimeout</code></td>
<td><code>1m</code></td>
<td>Registry <code>docker login</code> only.</td>
</tr>
<tr bgcolor="#f6f8fa">
<td><code>dockerComposePullTimeout</code></td>
<td><code>10m</code></td>
<td><code>docker compose pull</code> (proxy and apps).</td>
</tr>
<tr>
<td><code>dockerComposeUpTimeout</code></td>
<td><code>5m</code></td>
<td><code>docker compose up -d</code>.</td>
</tr>
<tr bgcolor="#f6f8fa">
<td><code>dockerComposeDownTimeout</code></td>
<td><code>2m</code></td>
<td><code>docker compose down</code> for unused apps. On failure the app dir is kept so the next cycle retries.</td>
</tr>
<tr>
<td><code>dockerPruneTimeout</code></td>
<td><code>2m</code></td>
<td><code>docker image prune -af</code>.</td>
</tr>
<tr bgcolor="#f6f8fa">
<td><code>dockerAPITimeout</code></td>
<td><code>30s</code></td>
<td>Docker Engine API container list used for start planning.</td>
</tr>
<tr>
<td><code>nginxCLITimeout</code></td>
<td><code>30s</code></td>
<td><code>docker exec … nginx -t</code> / <code>nginx -s reload</code>.</td>
</tr>
<tr bgcolor="#f6f8fa">
<td><code>domainPrecheckTimeout</code></td>
<td><code>15s</code></td>
<td>Domain check via <code>dig +short</code> before ACME.</td>
</tr>
<tr>
<td><code>certIssueTimeout</code></td>
<td><code>1m</code></td>
<td>ACME certificate obtain (HTTP-01). Timeout is a normal issuance error (retries / not rate-limit downgrade).</td>
</tr>
</tbody>
</table>

Example:

```properties
useExecLogs=true
certs.provider=letsencrypt
#certs.zerossl.AcmeKid=
#certs.zerossl.AcmeHmac=
gitTimeout=45s
dockerLoginTimeout=30s
dockerComposePullTimeout=15m
certIssueTimeout=5m
nginxCLITimeout=45s
```
