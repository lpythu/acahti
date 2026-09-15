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
	h := HandleConfig("secret")
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

func TestExpandSecretsList(t *testing.T) {
	got, err := Expand([]byte(`steps:
  publish:
    pipe: npm-publish@v1
    with:
      path: packages/pkg
      registry: https://acahti.example.com/api/packages/acme/npm
    secrets: [acahti_publish_token]
`))
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "acahti_publish_token") || !strings.Contains(text, "secrets:") {
		t.Fatalf("lost secrets list:\n%s", text)
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
