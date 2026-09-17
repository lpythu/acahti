package pipeline

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExpandHelm(t *testing.T) {
	src := []byte(`when:
  - event: [push]
labels:
  deploy: true
steps:
  deploy:
    pipe: helm@v1
    with:
      release: tm-console
      namespace: turbomesh
      take_ownership: true
      set: |
        workloads.tm-console.image.tag=${CI_COMMIT_SHA}
    secrets:
      KUBECONFIG: kubeconfig_office
  wait:
    pipe: wait-http@v1
    with:
      urls: |
        https://tm.dev.saidc.ai/
`)
	got, err := Expand(src)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if strings.Contains(text, "/home/saidc/.kube") || strings.Contains(text, "kubeconfig:") {
		t.Fatalf("host kubeconfig leftover:\n%s", text)
	}
	if !strings.Contains(text, "from_secret: kubeconfig_office") {
		t.Fatalf("missing from_secret:\n%s", text)
	}
	if strings.Contains(text, "pipe:") {
		t.Fatalf("pipe leftover:\n%s", text)
	}
	if !strings.Contains(text, "acahti-pipe helm") {
		t.Fatalf("missing helm dispatch:\n%s", text)
	}
	if !strings.Contains(text, "acahti-pipe wait-http") {
		t.Fatalf("missing wait-http dispatch:\n%s", text)
	}
	if !strings.Contains(text, "INPUT_RELEASE: tm-console") {
		t.Fatalf("missing INPUT_RELEASE:\n%s", text)
	}
	if !strings.Contains(text, "INPUT_TAKE_OWNERSHIP: \"true\"") && !strings.Contains(text, "INPUT_TAKE_OWNERSHIP: true") {
		t.Fatalf("missing take_ownership:\n%s", text)
	}
	if !strings.Contains(text, "image: bash") {
		t.Fatalf("missing image bash:\n%s", text)
	}
}

func TestExpandPassthroughCommands(t *testing.T) {
	src := []byte(`steps:
  build:
    image: bash
    commands:
      - echo hi
`)
	got, err := Expand(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "echo hi") {
		t.Fatalf("lost commands:\n%s", got)
	}
	if strings.Contains(string(got), "acahti-pipe") {
		t.Fatalf("unexpected dispatch:\n%s", got)
	}
}

func TestExpandRejectsUses(t *testing.T) {
	_, err := Expand([]byte("steps:\n  x:\n    uses: helm@v1\n"))
	if err == nil || !strings.Contains(err.Error(), "uses:") {
		t.Fatalf("err=%v", err)
	}
}

func TestExpandRejectsUnknown(t *testing.T) {
	_, err := Expand([]byte("steps:\n  x:\n    pipe: nope@v1\n"))
	if err == nil || !strings.Contains(err.Error(), "unknown pipe") {
		t.Fatalf("err=%v", err)
	}
}

func TestExpandRejectsMix(t *testing.T) {
	_, err := Expand([]byte("steps:\n  x:\n    pipe: helm@v1\n    commands: [echo]\n"))
	if err == nil || !strings.Contains(err.Error(), "cannot mix") {
		t.Fatalf("err=%v", err)
	}
}

func TestHandleConfigAuthAndExpand(t *testing.T) {
	h := HandleConfig("secret", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/hooks/pipeline-config", strings.NewReader(`{}`)))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("no auth %d", rr.Code)
	}
	body, _ := json.Marshal(configRequest{Configuration: []fileMeta{{
		Name: ".acahti/pipelines/ci.yaml",
		Data: "steps:\n  login:\n    pipe: docker-login@v1\n    with:\n      registry: harbor.example\n",
	}}})
	req := httptest.NewRequest(http.MethodPost, "/hooks/pipeline-config", strings.NewReader(string(body)))
	req.SetBasicAuth("pipe", "secret")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expand %d %s", rr.Code, rr.Body.String())
	}
	var resp configResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Configs) != 1 || !strings.Contains(resp.Configs[0].Data, "acahti-pipe docker-login") {
		t.Fatalf("resp=%+v", resp)
	}

	req = httptest.NewRequest(http.MethodPost, "/hooks/pipeline-config?token=secret", strings.NewReader(string(body)))
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("query token %d %s", rr.Code, rr.Body.String())
	}
}

func TestHandleConfigKeepsCIAndCD(t *testing.T) {
	h := HandleConfig("secret", nil)
	body, _ := json.Marshal(configRequest{
		Configuration: []fileMeta{
			{Name: "ci.yaml", Data: "when:\n  - event: [push]\nsteps:\n  x:\n    image: bash\n    commands: [true]\n"},
			{Name: "cd.office.yaml", Data: "when:\n  - event: [push]\n    branch: [dev]\ndepends_on: [ci]\nsteps:\n  x:\n    image: bash\n    commands: [true]\n"},
		},
	})
	var parsed configRequest
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatal(err)
	}
	parsed.Pipeline.Event = "push"
	parsed.Pipeline.Branch = "dev"
	parsed.Pipeline.Ref = "refs/heads/dev"
	body, _ = json.Marshal(parsed)
	req := httptest.NewRequest(http.MethodPost, "/hooks/pipeline-config", strings.NewReader(string(body)))
	req.SetBasicAuth("pipe", "secret")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("%d %s", rr.Code, rr.Body.String())
	}
	var resp configResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Configs) != 2 {
		t.Fatalf("want ci and cd.office, got %+v", resp)
	}
	names := resp.Configs[0].Name + " " + resp.Configs[1].Name
	if !strings.Contains(names, "ci.yaml") || !strings.Contains(names, "cd.office.yaml") {
		t.Fatalf("names=%s", names)
	}
	if !strings.Contains(resp.Configs[1].Data, "depends_on") {
		t.Fatalf("cd should keep depends_on ci:\n%s", resp.Configs[1].Data)
	}
}

func TestExpandSecretsList(t *testing.T) {
	got, err := Expand([]byte(`steps:
  login:
    pipe: docker-login@v1
    with:
      registry: harbor.example
    secrets: [harbor_password]
`))
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "from_secret: harbor_password") {
		t.Fatalf("missing from_secret:\n%s", text)
	}
	if !strings.Contains(text, "HARBOR_PASSWORD") {
		t.Fatalf("list secret not uppercased:\n%s", text)
	}
	if strings.Contains(text, "secrets:") {
		t.Fatalf("list secrets leftover:\n%s", text)
	}
	if strings.Contains(text, "INPUT_SECRETS") {
		t.Fatalf("secrets stringified:\n%s", text)
	}
}

func TestExpandSecretsMapOnCommands(t *testing.T) {
	got, err := Expand([]byte(`steps:
  upload:
    image: bash
    commands:
      - acahti-pipe oss-put
    secrets:
      OSS_ACCESS_KEY_ID: oss_access_key_id
`))
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "from_secret: oss_access_key_id") {
		t.Fatalf("missing from_secret:\n%s", text)
	}
	if strings.Contains(text, "secrets:") {
		t.Fatalf("map secrets leftover:\n%s", text)
	}
}

func TestExpandRejectsHostSecretFiles(t *testing.T) {
	_, err := Expand([]byte("steps:\n  x:\n    pipe: docker-login@v1\n    with:\n      registry: harbor.example\n      password_file: /root/.harbor/x\n"))
	if err == nil || !strings.Contains(err.Error(), "password_file") {
		t.Fatalf("err=%v", err)
	}
	_, err = Expand([]byte("steps:\n  x:\n    pipe: helm@v1\n    with:\n      kubeconfig: /home/saidc/.kube/office\n      release: a\n      namespace: b\n"))
	if err == nil || !strings.Contains(err.Error(), "kubeconfig") {
		t.Fatalf("err=%v", err)
	}
	_, err = Expand([]byte("steps:\n  x:\n    pipe: npm-publish@v1\n    with:\n      path: pkg\n      registry: https://example/npm\n      env_file: /root/.acahti.env\n"))
	if err == nil || !strings.Contains(err.Error(), "env_file") {
		t.Fatalf("err=%v", err)
	}
}

func TestExpandMergesExistingFromSecret(t *testing.T) {
	got, err := Expand([]byte(`steps:
  upload:
    image: bash
    commands:
      - acahti-pipe oss-put
    environment:
      KEEP: yes
      EXISTING:
        from_secret: keep_me
    secrets:
      OSS_ACCESS_KEY_ID: oss_access_key_id
`))
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "KEEP: yes") && !strings.Contains(text, "KEEP: \"yes\"") {
		t.Fatalf("lost KEEP:\n%s", text)
	}
	if !strings.Contains(text, "from_secret: keep_me") {
		t.Fatalf("flattened existing from_secret:\n%s", text)
	}
	if !strings.Contains(text, "from_secret: oss_access_key_id") {
		t.Fatalf("missing mapped secret:\n%s", text)
	}
}

func TestExpandArgosDashSecrets(t *testing.T) {
	got, err := Expand([]byte(`steps:
  e2e:
    pipe: argos@v1
    secrets:
      ARGOS_DASH_URL: argos_dash_url
      ARGOS_TOKEN: argos_token
    with:
      env: office
      selectors: pack:platform tag:cluster
`))
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "from_secret: argos_dash_url") || !strings.Contains(text, "from_secret: argos_token") {
		t.Fatalf("missing from_secret:\n%s", text)
	}
	if !strings.Contains(text, "acahti-pipe argos") {
		t.Fatalf("missing argos dispatch:\n%s", text)
	}
	if !strings.Contains(text, "INPUT_ENV: office") {
		t.Fatalf("missing env:\n%s", text)
	}
}

func TestExpandDockerGc(t *testing.T) {
	got, err := Expand([]byte(`steps:
  gc:
    pipe: docker-gc@v1
`))
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "acahti-pipe docker-gc") {
		t.Fatalf("missing docker-gc:\n%s", text)
	}
}

func TestExpandOciGc(t *testing.T) {
	got, err := Expand([]byte(`steps:
  gc:
    pipe: oci-gc@v1
    with:
      repos: |
        harbor.example/app
`))
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "acahti-pipe oci-gc") {
		t.Fatalf("missing oci-gc:\n%s", text)
	}
	if !strings.Contains(text, "INPUT_REPOS:") {
		t.Fatalf("missing INPUT_REPOS:\n%s", text)
	}
}

func TestExpandDockerLoginDoesNotInjectHarbor(t *testing.T) {
	got, err := Expand([]byte(`steps:
  login:
    pipe: docker-login@v1
    with:
      registry: saidc-registry.cn-hongkong.cr.aliyuncs.com
    secrets:
      DOCKER_USERNAME: acr_username
      DOCKER_PASSWORD: acr_password
`))
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if strings.Contains(text, "harbor_password") {
		t.Fatalf("harbor must not be auto-injected:\n%s", text)
	}
	if !strings.Contains(text, "from_secret: acr_password") {
		t.Fatalf("missing acr password:\n%s", text)
	}
}

func TestExpandIdentInjectsJob(t *testing.T) {
	got, err := ExpandIdent([]byte(`steps:
  build:
    pipe: docker-build@v1
    with:
      images: |
        app:dev
`), Ident{User: "lipeiyang", Token: "tok", RootURL: "https://acahti.example.com", Org: "acme"})
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "ACAHTI_USER: lipeiyang") {
		t.Fatalf("missing user:\n%s", text)
	}
	if !strings.Contains(text, "ACAHTI_TOKEN: tok") {
		t.Fatalf("missing token:\n%s", text)
	}
	if !strings.Contains(text, "ACAHTI_ROOT_URL: https://acahti.example.com") {
		t.Fatalf("missing root url:\n%s", text)
	}
	if !strings.Contains(text, "ACAHTI_ORG: acme") {
		t.Fatalf("missing org:\n%s", text)
	}
}

func TestHandleConfigIssuesJob(t *testing.T) {
	h := HandleConfig("secret", func(author, repo, sha string) Ident {
		if author != "acahti" || repo != "saidc/tm-web" || sha != "abc" {
			t.Fatalf("issue args %q %q %q", author, repo, sha)
		}
		return Ident{User: "lipeiyang", Token: "tok"}
	})
	var cfg configRequest
	cfg.Configuration = []fileMeta{{
		Name: ".acahti/pipelines/ci.yaml",
		Data: "steps:\n  build:\n    pipe: docker-build@v1\n    with:\n      images: |\n        app:dev\n",
	}}
	cfg.Pipeline.Author = "acahti"
	cfg.Pipeline.Commit = "abc"
	cfg.Repo.Owner = "saidc"
	cfg.Repo.Name = "tm-web"
	body, _ := json.Marshal(cfg)
	req := httptest.NewRequest(http.MethodPost, "/hooks/pipeline-config", strings.NewReader(string(body)))
	req.SetBasicAuth("pipe", "secret")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code %d %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "ACAHTI_USER: lipeiyang") {
		t.Fatalf("body=%s", rr.Body.String())
	}

	office := configRequest{}
	office.Configuration = []fileMeta{{
		Name: ".acahti/pipelines/cd.office.yaml",
		Data: "steps:\n  deploy:\n    pipe: helm@v1\n    with:\n      release: a\n      namespace: b\n",
	}}
	office.Pipeline.Author = "acahti"
	office.Pipeline.Commit = "abc"
	office.Repo.Owner = "saidc"
	office.Repo.Name = "tm-web"
	body, _ = json.Marshal(office)
	req = httptest.NewRequest(http.MethodPost, "/hooks/pipeline-config", strings.NewReader(string(body)))
	req.SetBasicAuth("pipe", "secret")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("office %d %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "deploy-office-saidc-tm-web") {
		t.Fatalf("missing concurrency: %s", rr.Body.String())
	}
}

func TestExpandFileInjectsOfficeConcurrency(t *testing.T) {
	got, err := ExpandFile(".acahti/pipelines/cd.office.yaml", []byte(`steps:
  deploy:
    pipe: helm@v1
    with:
      release: app
      namespace: default
`), Ident{Repo: "saidc/tm-web"})
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "group: deploy-office-saidc-tm-web") || !strings.Contains(text, "limit: 1") {
		t.Fatalf("missing concurrency:\n%s", text)
	}
}

func TestExpandFileInjectsHkAndPkg(t *testing.T) {
	hk, err := ExpandFile("cd.hk.yaml", []byte("steps:\n  x:\n    image: bash\n    commands: [true]\n"), Ident{Repo: "saidc/exhub"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(hk), "group: deploy-hk-saidc-exhub") {
		t.Fatalf("hk:\n%s", hk)
	}
	pkg, err := ExpandFile("pkg.yaml", []byte("steps:\n  x:\n    image: bash\n    commands: [true]\n"), Ident{Repo: "saidc/saidc-ui"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pkg), "group: pkg-saidc-saidc-ui") {
		t.Fatalf("pkg:\n%s", pkg)
	}
}

func TestExpandFileKeepsYAMLConcurrency(t *testing.T) {
	got, err := ExpandFile("cd.office.yaml", []byte(`concurrency:
  limit: 2
  group: custom
steps:
  x:
    image: bash
    commands: [true]
`), Ident{})
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "group: custom") || strings.Contains(text, "deploy-office") {
		t.Fatalf("%s", text)
	}
}

func TestExpandFileSkipsCI(t *testing.T) {
	got, err := ExpandFile("ci.yaml", []byte("steps:\n  x:\n    image: bash\n    commands: [true]\n"), Ident{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "concurrency:") {
		t.Fatalf("ci should not get concurrency:\n%s", got)
	}
}
