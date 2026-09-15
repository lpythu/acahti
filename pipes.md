# Official pipes

A **pipeline** is one run. Each YAML file in `.acahti/pipelines/` is a **job** in that run. A **step** is a named block in a job. A **pipe** is an official reusable implementation a step can call.

Acahti expands `pipe:` before the Runner executes. The Runner runs `acahti-pipe <name>`. Do not vendor `.acahti/scripts`. Do not write `uses:`.

A step is either `pipe:` + `with:` or raw `commands:` (one-offs stay in the repo). `when`, `depends_on`, and `labels` pass through.

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

Secret names are lowercase `snake_case`. Woodpecker injects them as uppercase env. A step must opt in.

List when the secret name **is** the env the pipe wants:

```yaml
secrets: [acahti_publish_token]
```

Map when they differ:

```yaml
secrets:
  KUBECONFIG: kubeconfig_office
  DOCKER_PASSWORD: harbor_password
```

Do not put `env_file`, `token_file`, `password_file`, `auth_file`, or a kubeconfig **path** in `with:`.

Org catalog (Owners put once under Admin → Pipeline secrets):

| Secret | Pipe env |
|---|---|
| `harbor_password` | `DOCKER_PASSWORD` (office CD) |
| `acr_username` / `acr_password` | `DOCKER_USERNAME` / `DOCKER_PASSWORD` (HK CD) |
| `kubeconfig_office` / `kubeconfig_hk` | `KUBECONFIG` |
| `acahti_publish_token` | `ACAHTI_PUBLISH_TOKEN` |
| `npm_token` | `NPM_TOKEN` |
| `codeup_netrc` | `CODEUP_NETRC` (Go private modules still fetched from Codeup) |
| `oss_access_key_id` / `oss_access_key_secret` | `OSS_ACCESS_KEY_ID` / `OSS_ACCESS_KEY_SECRET` |
| `argos_dash` | `ARGOS_DASH` |

Only org/repo admins can list or put secrets. Members cannot see names. Values are never returned.

## Catalog

### docker-login

`with:` `registry` (required), `username` (or env `DOCKER_USERNAME`). Password is env `DOCKER_PASSWORD`.

### docker-build

`with:` `images` (required). One image per line. First field is the primary tag. Optional `also=` extra tags (comma-separated), `context=` (default `.`), `file=` Dockerfile, other `KEY=VAL` as `--build-arg`. Optional `push` (default `true`); `push: false` is `--load` only (CI). BuildKit secrets: `NPM_TOKEN` → `id=npm_token`, `CODEUP_NETRC` → `id=codeup_netrc`. Builds with `--pull --provenance=false --load`, then pushes primary and `also=` tags when `push` is true.

OCI tags: office CD `dev-${CI_COMMIT_SHA}`; HK CD `${CI_COMMIT_TAG#v}` (git tag `vX.Y.Z` → `X.Y.Z`). `latest` is a pointer at the same digest. `${CI_COMMIT_TAG#v}` is expanded in the pipe (Woodpecker only interpolates `${CI_COMMIT_TAG}`).

### oci-gc

`with:` `repos` (required, one OCI repository per line, no tag). Optional `keep` (non-`latest` tags to keep, default 2). Always keeps `latest`. Uses `crane` already on the Runner (same class as docker/helm; `agent.sh` does not install it). Auth is the docker login already done in the job. Run after helm succeeds so retained tags are deployed ones.

### helm

`with:` `release`, `namespace` (required). Optional `chart` (default `chart`), `files` (`-f` paths), `set` (`--set` lines), `timeout` (default `5m`), `take_ownership: true`, `jump` (SSH host on the Runner; HK ACK is VPC-only so `thk`). Env `KUBECONFIG` must be the kubeconfig **document** from a secret. The Runner has helm and crane (`agent.sh`).

### wait-http

`with:` `urls` (newline or semicolon). Polls until the status is not 502/503/504/000. Optional `timeout` seconds (default 180).

### argos

`with:` `env`, `selectors` (semicolon-separated `argos run` invocations). Optional env `ARGOS_DASH` (dash file content) → `--dash`.

### npm-publish

Build the package first (`commands:`). `with:` `path` (package dir with `dist/`), `registry` (packages npm URL). Token is env `ACAHTI_PUBLISH_TOKEN`. Optional `origin`, `org`, `user` (derived from `registry` when omitted).

### pypi-publish

`with:` `registry` (uv `--publish-url`). Token is env `ACAHTI_PUBLISH_TOKEN`. Runs `uv build` then `uv publish`.

### oss-put

`with:` `endpoint`, `bucket`, `prefix`, `files` (workspace-relative, one per line). Env `OSS_ACCESS_KEY_ID` and `OSS_ACCESS_KEY_SECRET`.

## Examples

CI verifies the image (`push: false`). It does not publish. Harbor login is only for `--pull` of private bases.

```yaml
steps:
  login:
    pipe: docker-login@v1
    with:
      registry: harbor.example
      username: 'robot$$user'
    secrets:
      DOCKER_PASSWORD: harbor_password
  build:
    pipe: docker-build@v1
    with:
      push: false
      images: |
        app:${CI_COMMIT_SHA} APP=web
    secrets:
      NPM_TOKEN: npm_token
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
      DOCKER_PASSWORD: harbor_password
  build:
    pipe: docker-build@v1
    with:
      images: |
        harbor.example/app:dev-${CI_COMMIT_SHA} also=harbor.example/app:latest APP=web
    secrets:
      NPM_TOKEN: npm_token
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

npm package (Node is a container, not a Runner install):

```yaml
steps:
  build:
    image: bash
    commands:
      - |
        docker run --rm --network=host -v "$PWD:/app" -w /app harbor.example/library/node:22-alpine sh -c "corepack enable && pnpm install --frozen-lockfile && pnpm --filter @scope/pkg build"
  publish:
    pipe: npm-publish@v1
    with:
      path: packages/pkg
      registry: https://acahti.example.com/api/packages/acme/npm
      image: harbor.example/library/node:22-alpine
    secrets: [acahti_publish_token]
```
