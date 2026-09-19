package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"acahti/internal/auth"
	"acahti/internal/catalog"
	"acahti/internal/config"
	"acahti/internal/forgejo"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func testHTTPServer(t *testing.T) (*httptest.Server, *auth.Service) {
	t.Helper()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/user" {
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"login": r.Header.Get("Sudo")})
	}))
	t.Cleanup(upstream.Close)
	a := auth.New([]byte("test-only-signing-key"), "admin")
	cat := catalog.New(config.Config{RootURL: "https://instance.example", Domain: "instance.example", Org: "acme", Version: "test"}, forgejo.New(upstream.URL, "test-only"), nil, nil)
	s := New(config.Config{RootURL: "https://instance.example", Domain: "instance.example", Org: "acme", Version: "test"}, a, cat)
	hs := httptest.NewServer(s)
	t.Cleanup(hs.Close)
	return hs, a
}

func request(t *testing.T, url, token, method, body string) (int, http.Header, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, res.Header, raw
}

func TestHTTPHandshake(t *testing.T) {
	hs, a := testHTTPServer(t)
	token := a.Issue("alice")
	status, headers, raw := request(t, hs.URL, token, "POST", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`)
	if status != 200 {
		t.Fatalf("initialize: %d %s", status, raw)
	}
	var init struct{ Result sdk.InitializeResult }
	if err := json.Unmarshal(raw, &init); err != nil {
		t.Fatal(err)
	}
	if init.Result.ServerInfo.Name != "acahti" || init.Result.ServerInfo.Version != "test" || !strings.Contains(init.Result.Instructions, "instance.example") {
		t.Fatalf("initialize: %s", raw)
	}
	if headers.Get("Mcp-Session-Id") != "" {
		t.Fatal("expected stateless transport")
	}
	status, _, raw = request(t, hs.URL, token, "POST", `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if status != 202 || len(raw) != 0 {
		t.Fatalf("notification: %d %s", status, raw)
	}
	status, _, raw = request(t, hs.URL, token, "POST", `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if status != 200 || !strings.Contains(string(raw), `"whoami"`) || !strings.Contains(string(raw), `"inbox"`) || !strings.Contains(string(raw), `"secret_list"`) {
		t.Fatalf("tools: %d %s", status, raw)
	}
	for _, method := range []string{"GET", "DELETE"} {
		status, _, _ = request(t, hs.URL, token, method, "")
		if status != 405 {
			t.Fatalf("%s: %d", method, status)
		}
	}
}

func TestHTTPHandshakePublic(t *testing.T) {
	hs, _ := testHTTPServer(t)
	status, _, raw := request(t, hs.URL, "", "POST", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`)
	if status != 200 || !strings.Contains(string(raw), `"acahti"`) {
		t.Fatalf("public initialize: %d %s", status, raw)
	}
	status, _, raw = request(t, hs.URL, "", "POST", `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if status != 200 || !strings.Contains(string(raw), `"whoami"`) {
		t.Fatalf("public tools/list: %d %s", status, raw)
	}
}

func TestHTTPAuthentication(t *testing.T) {
	hs, a := testHTTPServer(t)
	a.TTL = -time.Minute
	call := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"whoami","arguments":{}}}`
	for _, token := range []string{"", "invalid", a.Issue("expired")} {
		status, headers, _ := request(t, hs.URL, token, "POST", call)
		if status != 401 || !strings.Contains(headers.Get("WWW-Authenticate"), "oauth-protected-resource") {
			t.Fatalf("auth: %d %v", status, headers)
		}
		wantInvalid := token != ""
		hasInvalid := strings.Contains(headers.Get("WWW-Authenticate"), "invalid_token")
		if hasInvalid != wantInvalid {
			t.Fatalf("auth invalid_token=%v want %v token=%q header=%s", hasInvalid, wantInvalid, token, headers.Get("WWW-Authenticate"))
		}
	}
}

type authenticatedTransport struct{ token string }

func (a authenticatedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	clone := r.Clone(r.Context())
	clone.Header.Set("Authorization", "Bearer "+a.token)
	return http.DefaultTransport.RoundTrip(clone)
}

func TestSDKClientIdentityIsolation(t *testing.T) {
	hs, a := testHTTPServer(t)
	for _, login := range []string{"alice", "bob", "carol"} {
		t.Run(login, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			client := sdk.NewClient(&sdk.Implementation{Name: "integration-test", Version: "1"}, nil)
			session, err := client.Connect(ctx, &sdk.StreamableClientTransport{Endpoint: hs.URL, HTTPClient: &http.Client{Transport: authenticatedTransport{a.Issue(login)}}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			for range 3 {
				result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "whoami", Arguments: map[string]any{}})
				if err != nil {
					t.Fatal(err)
				}
				if result.IsError {
					t.Fatalf("whoami failed: %+v", result)
				}
				var identity struct {
					Login string `json:"login"`
				}
				if err := json.Unmarshal([]byte(result.Content[0].(*sdk.TextContent).Text), &identity); err != nil {
					t.Fatal(err)
				}
				if identity.Login != login {
					t.Fatalf("identity crossed requests: want %s, got %s", login, identity.Login)
				}
			}
			if result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "repo_get", Arguments: map[string]any{}}); err == nil && !result.IsError {
				t.Fatal("missing required arguments accepted")
			}
			result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "repo_get", Arguments: map[string]any{"owner": "acme", "name": "missing"}})
			if err != nil || !result.IsError {
				t.Fatalf("business failure should be a tool error: result=%+v err=%v", result, err)
			}
		})
	}
}

func TestHTTPCrossOrigin(t *testing.T) {
	hs, a := testHTTPServer(t)
	req, err := http.NewRequest("POST", hs.URL, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+a.Issue("alice"))
	req.Header.Set("Origin", "https://untrusted.example")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-origin status: %d", res.StatusCode)
	}
}
