# Official pipes

A **pipeline** is one run. Each YAML file in `.acahti/pipelines/` is a **job** in that run. A **step** is a named block in a job. A **pipe** is an official reusable implementation a step can call.

Acahti expands `pipe:` before the Runner executes. The Runner runs `acahti-pipe <name>`. Do not vendor `.acahti/scripts`. Do not write `uses:`.

A step is either `pipe:` + `with:` or raw `commands:` (one-offs stay in the repo). `when`, `depends_on`, and `labels` pass through. Expand injects Woodpecker `concurrency` when the job file omits it: `cd.office` → group `deploy-office-<owner>-<repo>` (limit 1), `cd.hk` → `deploy-hk-…`, `pkg` → `pkg-…`. Same repo still serializes; different repos run in parallel. YAML `concurrency:` wins. CI files are unlimited except Runner capacity (`WOODPECKER_MAX_WORKFLOWS`, from host nproc/memory unless pinned).

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

This train only first-party pipes: `pipe: <name>@v1`. Registry URLs, image names, wait URLs, and argos selectors belong in the repo YAML.

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
| `acr_username` / `acr_password` | `DOCKER_USERNAME` / `DOCKER_PASSWORD` (hk) |
| `kubeconfig_office` / `kubeconfig_hk` | `KUBECONFIG` |
| `codeup_netrc` | `CODEUP_NETRC` (Go modules still fetched from Codeup git) |
| `oss_access_key_id` / `oss_access_key_secret` | `OSS_ACCESS_KEY_ID` / `OSS_ACCESS_KEY_SECRET` |
| `argos_dash` | `ARGOS_DASH` |

Only org/repo admins can list or put secrets. Repo Secrets shows the effective set. Members cannot see names. Values are never returned.

## Catalog

### docker-login

`with:` `registry` (required), `username` (or env `DOCKER_USERNAME`). Password is env `DOCKER_PASSWORD`.

### docker-build

`with:` `images` (required). One image per line. First field is the primary tag. Optional `also=` extra tags (comma-separated), `context=` (default `.`), `file=` Dockerfile, other `KEY=VAL` as `--build-arg` (CI names `BASE_IMAGE=harbor.saidc/base/…` or the ACR library equivalent; Dockerfiles stay Harbor-free). Optional `push` (default `true`); `push: false` is `--output=type=cacheonly` (CI, no `--load`). Job identity is forwarded as BuildKit secrets `id=acahti_user` / `id=acahti_token` (`ACAHTI_USER` / `ACAHTI_TOKEN`). The pipe does not write `.npmrc` or `.netrc`. `CODEUP_NETRC` → BuildKit `id=codeup_netrc`. Default builder `acahti` (`docker-container`, host network). Builds with `--pull --provenance=false`; `push: true` is `buildx --push` for primary and `also=` tags. Empty `images:` lines are skipped. After a successful build the pipe runs `docker-gc`.

OCI tags: office CD `dev-${CI_COMMIT_SHA}`; HK CD `${CI_COMMIT_TAG#v}` (git tag `vX.Y.Z` → `X.Y.Z`). `latest` is a pointer at the same digest. `${CI_COMMIT_TAG#v}` is expanded in the pipe (`${CI_COMMIT_TAG}` is interpolated by the runner).

### oci-gc

`with:` `repos` (required, one OCI repository per line, no tag). Optional `keep` (non-`latest` tags to keep, default 2). Always keeps `latest`. Uses `crane` already on the Runner (same class as docker/helm; `agent.sh` does not install it). Auth is the docker login already done in the job. Run after helm succeeds so retained tags are deployed ones.

### docker-gc

No `with:` required. Caps the Runner’s local BuildKit cache (`acahti` builder `--keep-storage=16GB`, `4GB` when `/` is ≥85% full), deletes other buildx builders, dangling images, stale `/tmp/woodpecker-local-*` (>6h), and leftover compose/云效 trees. Does not `docker system prune -a`. `docker-build` runs it after a successful build; `agent.sh` also installs an hourly systemd timer. Does not touch Harbor/ACR — that is `oci-gc`.

### helm

`with:` `release`, `namespace` (required). Optional `chart` (default `chart`), `files` (`-f` paths), `set` (`--set` lines), `timeout` (default `5m`), `take_ownership: true`, `jump` (SSH host on the Runner; HK ACK is VPC-only so `thk`). Env `KUBECONFIG` must be the kubeconfig **document** from a secret. The Runner has helm and crane (`agent.sh`).

### wait-http

`with:` `urls` (newline or semicolon). Polls until the status is not 502/503/504/000. Optional `timeout` seconds (default 180).

### argos

`with:` `env`, `selectors` (semicolon-separated `argos run` invocations). Optional env `ARGOS_DASH` (dash file content) → `--dash`.

### npm-publish

`with:` `path` (package dir), `registry` (Acahti packages npm URL). Optional `image` (default `node:22-alpine`). The pipe writes job-identity npm auth in the container (not the workspace), `pnpm install --frozen-lockfile --ignore-scripts`, `pnpm --filter <package.json name> build`, then `npm pack` and PUT. YAML does not `docker run` or write `.npmrc`. Optional `origin`, `org`, `user` (derived from `registry` when omitted).

### pypi-publish

`with:` `registry` (uv `--publish-url`). Optional `path` (default `.`), `image` (default `python:3.12-slim-bookworm`). The pipe runs `uv build` then `uv publish` in that image (installs uv if missing). Job identity is `UV_PUBLISH_*` and `UV_INDEX_<NAME>_*` for `[[tool.uv.index]]` entries with `authenticate = always`. YAML does not `docker run` or name a token.

### oss-put

`with:` `endpoint`, `bucket`, `prefix`, `files` (workspace-relative, one per line). Env `OSS_ACCESS_KEY_ID` and `OSS_ACCESS_KEY_SECRET`.

## Examples

CI verifies the image (`push: false`, cache-only). It does not publish. Harbor login is only for `--pull` of private bases.

```yaml
steps:
  login:
    pipe: docker-login@v1
    with:
      registry: harbor.example
      username: 'robot$$user'
    secrets:
      DOCKER_USERNAME: harbor_username
      DOCKER_PASSWORD: harbor_password
  build:
    pipe: docker-build@v1
    with:
      push: false
      images: |
        app:${CI_COMMIT_SHA} APP=web
```

Office CD (push `dev`) publishes Harbor `dev-${CI_COMMIT_SHA}` + `latest`, helm pin is that tag, then GC keeps 3 tags including `latest`. HK CD (tag on `test`) is the same shape with ACR login, image tag `${CI_COMMIT_TAG#v}`, `kubeconfig_hk`, and `jump: thk` (ACK API is VPC-only).

```yaml
steps:
  login:
    pipe: docker-login@v1
    with:
      registry: harbor.example
      username: 'robot$$user'
    secrets:
      DOCKER_USERNAME: harbor_username
      DOCKER_PASSWORD: harbor_password
  build:
    pipe: docker-build@v1
    with:
      images: |
        harbor.example/app:dev-${CI_COMMIT_SHA} also=harbor.example/app:latest APP=web
  deploy:
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
    pipe: oci-gc@v1
    with:
      repos: |
        harbor.example/app
  e2e:
    pipe: argos@v1
    with:
      env: office
      selectors: pack:platform tag:app
    secrets:
      ARGOS_DASH: argos_dash
```

Omit `wait` when there is no public URL. Repository stays in chart values — do not `--set` it.

npm / PyPI package. Harbor image → `docker-login` first. The publish pipe builds and uploads.

```yaml
steps:
  login:
    pipe: docker-login@v1
    with:
      registry: harbor.example
    secrets:
      DOCKER_USERNAME: harbor_username
      DOCKER_PASSWORD: harbor_password
  publish:
    depends_on:
      - login
    pipe: npm-publish@v1
    with:
      path: packages/pkg
      registry: https://acahti.example.com/api/packages/acme/npm
      image: harbor.example/library/node:22-alpine
```

```yaml
steps:
  publish:
    pipe: pypi-publish@v1
    with:
      registry: https://acahti.example.com/api/packages/acme/pypi
      image: harbor.example/base/saidc-uv:0.12.0
```
