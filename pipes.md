# Official pipes

A **pipeline** is one run. Each YAML file in `.acahti/pipelines/` is a **job** in that run. A **step** is a named block in a job. A **pipe** is an official reusable implementation a step can call.

Acahti expands `pipe:` before the Runner executes. The Runner runs `acahti-pipe <name>`. Do not vendor `.acahti/scripts`. Do not write `uses:`.

A step is either `pipe:` + `with:` or raw `commands:` (one-offs stay in the repo). `when`, `depends_on`, and `labels` pass through. If YAML declares `concurrency:`, expand repo-scopes `group` (`deploy` + repo `saidc/tm-web` → `deploy-saidc-tm-web`) so Woodpecker’s cluster-wide lock does not serialize unrelated repos. Omit `concurrency` to run freely. Job graph (`depends_on`, filenames) belongs in the repo YAML. The config hook orders those files by `depends_on` (parents before children, same rank by name). Woodpecker still numbers workflows by filename, so the catalog attaches YAML `depends_on` and paints pills left to right.

In `commands:`, write `$IMAGE` (shell). `${IMAGE}` is emptied by the runner before the step starts; `${CI_COMMIT_SHA}` is job context and is expanded.

Credentials are **pipeline secrets** (control plane). The Runner has no `/root/.harbor`, kubeconfig files, or npm tokens. YAML names secrets; values never go in git.

```yaml
steps:
  deploy:
    pipe: helm@v1
    with:
      release: tm-console
      namespace: turbomesh
      chart: chart
      files: |
        chart/values.yaml
        chart/values-office.yaml
      set: |
        workloads.app.image.tag=dev-${CI_COMMIT_SHA}
      take_ownership: true
    secrets:
      KUBECONFIG: kubeconfig_office
```

This train only first-party pipes: `pipe: <name>@v1`. Image names, wait URLs, and tool argv belong in the repo YAML. `npm-publish-acahti` / `pypi-publish-acahti` publish only to this Acahti org — do not pass a registry URL.

## Secrets

Secret names are lowercase `snake_case`. A step must opt in. Island-external only.

List when the secret name **is** the env the pipe wants. Map when they differ:

```yaml
secrets:
  KUBECONFIG: kubeconfig_office
  DOCKER_USERNAME: harbor_username
  DOCKER_PASSWORD: harbor_password
```

Do not put `env_file`, `token_file`, `password_file`, `auth_file`, or a kubeconfig **path** in `with:`.

Acahti npm/pypi install and publish use the triggering user's identity. Do not name an npm or publish token.

Org catalog (Owners put once under Admin → Org secrets). Harbor (office) and ACR (hk) are a pair:

| Secret | Pipe env |
|---|---|
| `harbor_username` / `harbor_password` | `DOCKER_USERNAME` / `DOCKER_PASSWORD` (office) |
| `acr_username` / `acr_password` | Aliyun AccessKey used by `docker-login` to mint ACR `GetAuthorizationToken` (bj-test / `cd.hk`) |
| `kubeconfig_office` / `kubeconfig_bj` | `KUBECONFIG` |
| `codeup_netrc` | `CODEUP_NETRC` (Go modules still fetched from Codeup git) |
| `oss_access_key_id` / `oss_access_key_secret` | `OSS_ACCESS_KEY_ID` / `OSS_ACCESS_KEY_SECRET` |
| `argos_dash_url` / `argos_token` | `ARGOS_DASH_URL` / `ARGOS_TOKEN` |

Only org/repo admins can list or put secrets. Repo Secrets shows the effective set. Members cannot see names. Values are never returned.

## Catalog

### docker-login

`with:` `registry` (required hostname), optional `http: true` (HTTP/insecure BuildKit; also accepted as `registry: http://host`). `username` (or env `DOCKER_USERNAME`). Password is env `DOCKER_PASSWORD`. Acahti does not hardcode Harbor or ACR — YAML names the host. `http: true` merges that host into the runner’s `buildkitd.toml` and recreates the shared `acahti` builder. HTTPS registries omit `http`. One registry per `docker-login` step.

### docker-build

`with:` `images` (required). One image per line. First field is the primary tag. Optional `also=` extra tags (comma-separated), `context=` (default `.`), `file=` Dockerfile, other `KEY=VAL` as `--build-arg`. Product Dockerfiles default `BASE_IMAGE` to HK ACR `saidc-bj-registry.cn-beijing.cr.aliyuncs.com/base/…`; office CD overrides the host to `harbor.saidc`. Optional `push` (default `true`); `push: false` is `--output=type=cacheonly` (CI, no `--load`). Job identity is written as BuildKit secrets `id=npmrc` and `id=netrc` (HTTP Basic from `ACAHTI_USER` / `ACAHTI_TOKEN`). Dockerfiles mount `id=npmrc` at `/root/.npmrc` and `id=netrc` at `/root/.netrc`. YAML does not name npm tokens. `CODEUP_NETRC` → BuildKit `id=codeup_netrc`. Default builder `acahti` (`docker-container`, host network). HTTP/insecure registries are declared on `docker-login` (`http: true`), not inferred by Acahti. Builds always `--pull --provenance=false`. Harbor `base` and `library` are anonymous-pull so CI can `--pull` without `docker-login`. `push: true` is `buildx --push` for primary and `also=` tags. `push: false` is cache-only (CI, no `--load`). Empty `images:` lines are skipped. Does not run `docker-gc` (hourly timer on the Runner does). Prints a one-line heartbeat every 60s while `buildx` runs so a live build is not reaped as a silent Woodpecker step.

OCI tags: office CD `dev-${CI_COMMIT_SHA}`; HK CD `${CI_COMMIT_TAG#v}` (git tag `vX.Y.Z` → `X.Y.Z`). `latest` is a pointer at the same digest. `${CI_COMMIT_TAG#v}` is expanded in the pipe (`${CI_COMMIT_TAG}` is interpolated by the runner).

### oci-gc

`with:` `repos` (required, one OCI repository per line, no tag). Optional `keep` (non-`latest` tags to keep, default 2). Always keeps `latest`. Uses `crane` already on the Runner (same class as docker/helm; `agent.sh` does not install it). Auth is the docker login already done in the job. Run after helm succeeds so retained tags are deployed ones.

### docker-gc

No `with:` required. Hourly timer only (`agent.sh`). Caps the `acahti` builder with unused-layer `--keep-storage` (min of avail−headroom and 25% of disk; headroom max(20GB, 15% of disk); clamp 8GB..64GB; `ACAHTI_BUILDKIT_KEEP` overrides). Does not `--all` (cache mounts stay). Deletes other buildx builders, dangling images, and stale `/tmp/woodpecker-local-*` (>6h). Does not `docker system prune -a`. Does not touch Harbor/ACR — that is `oci-gc`.

### helm

`with:` `release`, `namespace` (required). Optional `chart` (default `chart`), `files` (`-f` paths), `set` (`--set` lines), `timeout` (default `5m`), `take_ownership: true`, `jump` (SSH host on the Runner; HK ACK is VPC-only so `thk`). Env `KUBECONFIG` must be the kubeconfig **document** from a secret. The Runner has helm and crane (`agent.sh`).

### wait-http

`with:` `urls` (newline or semicolon). Polls until the status is not 502/503/504/000. Optional `timeout` seconds (default 180).

### uv

`with:` `run` (required, one command per line). Optional `project` (directory with `pyproject.toml`, default `.`). Installs `uv` if missing, then `uv run --default-index https://pypi.org/simple --project <dir> -- bash -c <line>` in a job-local venv (`UV_PROJECT_ENVIRONMENT`). Ignores inherited `UV_INDEX_URL` (stale mirrors omit new PyPI versions). Product YAML supplies the command (for example `argos run all --env office --dash`). Secrets become step env; `--dash` with no file reads `ARGOS_DASH_URL` / `ARGOS_TOKEN`. Argos CLI in CI reads Woodpecker `CI_*` and `ACAHTI_ROOT_URL` so the dash run links back to this pipeline; it prints `argos <sid>` and `dash {url}` (`{ARGOS_DASH_URL}/runs/{sid}`). Acahti shows that URL on the pipeline list and detail when the job name is `e2e.*` or a step is named `e2e`.

### npm-publish-acahti

Acahti org npm only. `with:` `path` (package dir). Optional `image` (default `node:22-alpine`). The pipe writes job-identity npm auth for both the public host and the LAN gateway, then `pnpm install --frozen-lockfile --ignore-scripts`, `pnpm --filter <package.json name> build`, `npm pack`, and PUT to the LAN gateway (`WOODPECKER_SERVER` host `:8080`, `Host` = `ACAHTI_ROOT_URL`). HTTP 409 (version already on the index) is success. YAML does not name a registry, `docker run`, or write `.npmrc`.

### pypi-publish-acahti

Acahti org PyPI only. Optional `path` (default `.`), `image` (default `python:3.12-slim-bookworm`). The pipe runs `uv build` in that image (installs uv if missing) then POST twine multipart to the LAN gateway (`/api/packages/<org>/pypi`). HTTP 409 (version already on the index) is success. YAML does not name a registry, `docker run`, or a token.

### oss-put

`with:` `endpoint`, `bucket`, `prefix`, `files` (workspace-relative, one per line). Env `OSS_ACCESS_KEY_ID` and `OSS_ACCESS_KEY_SECRET`.

## Examples

CI verifies the image (`push: false`, cache-only). It does not login or publish.

```yaml
steps:
  build:
    pipe: docker-build@v1
    with:
      push: false
      images: |
        app:${CI_COMMIT_SHA} APP=web BASE_IMAGE=harbor.example/base/saidc-node:22
```

CD jobs declare a deploy lock. Office YAML names Harbor for bases and products. HK YAML names ACR for both — do not pull Harbor from an HK job.

```yaml
concurrency:
  limit: 1
  group: deploy
steps:
  login:
    pipe: docker-login@v1
    with:
      registry: harbor.example
      http: true
    secrets:
      DOCKER_USERNAME: harbor_username
      DOCKER_PASSWORD: harbor_password
  build:
    depends_on: [login]
    pipe: docker-build@v1
    with:
      images: |
        harbor.example/app:dev-${CI_COMMIT_SHA} also=harbor.example/app:latest APP=web BASE_IMAGE=harbor.example/base/saidc-node:22
  deploy:
    depends_on: [build]
    pipe: helm@v1
    with:
      release: app
      namespace: default
      files: |
        chart/values.yaml
        chart/values-office.yaml
      set: |
        workloads.app.image.tag=dev-${CI_COMMIT_SHA}
      take_ownership: true
    secrets:
      KUBECONFIG: kubeconfig_office
  gc:
    depends_on: [deploy]
    pipe: oci-gc@v1
    with:
      repos: |
        harbor.example/app
```

A later job can `depends_on` CD and run tools via `uv`:

```yaml
depends_on: [cd.office]
steps:
  e2e:
    pipe: uv@v1
    secrets:
      ARGOS_DASH_URL: argos_dash_url
      ARGOS_TOKEN: argos_token
    with:
      project: .acahti/argos
      run: |
        argos run all --env office --dash
```

Omit `wait` when there is no public URL. Repository stays in chart values — do not `--set` it.

npm / PyPI package. Private image → `docker-login` first. The publish pipe builds and uploads.

```yaml
steps:
  login:
    pipe: docker-login@v1
    with:
      registry: harbor.example
      http: true
    secrets:
      DOCKER_USERNAME: harbor_username
      DOCKER_PASSWORD: harbor_password
  publish:
    depends_on:
      - login
    pipe: npm-publish-acahti@v1
    with:
      path: packages/pkg
      image: harbor.example/library/node:22-alpine
```

```yaml
steps:
  login:
    pipe: docker-login@v1
    with:
      registry: harbor.example
      http: true
    secrets:
      DOCKER_USERNAME: harbor_username
      DOCKER_PASSWORD: harbor_password
  publish:
    depends_on:
      - login
    pipe: pypi-publish-acahti@v1
    with:
      image: harbor.example/base/python:3.12-slim-bookworm
```
