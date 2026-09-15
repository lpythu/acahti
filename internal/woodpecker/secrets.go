package woodpecker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"acahti/internal/page"
)

var DefaultSecretEvents = []string{"push", "tag", "manual"}

type Secret struct {
	Name   string   `json:"name"`
	Events []string `json:"events,omitempty"`
}

type kernelSecret struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Value       string   `json:"value,omitempty"`
	Events      []string `json:"events"`
	Images      []string `json:"images"`
	PluginsOnly bool     `json:"plugins_only"`
}

type Org struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func publicSecret(s kernelSecret) Secret {
	return Secret{Name: s.Name, Events: s.Events}
}

func secretBody(name, value string, events []string) map[string]any {
	if len(events) == 0 {
		events = DefaultSecretEvents
	}
	body := map[string]any{
		"name":         name,
		"events":       events,
		"images":       []string{},
		"plugins_only": false,
	}
	if value != "" {
		body["value"] = value
	}
	return body
}

func (c *Client) orgKey(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("empty org")
	}
	if _, err := strconv.ParseInt(name, 10, 64); err == nil {
		return name, nil
	}
	c.mu.Lock()
	id, ok := c.ids["org:"+name]
	c.mu.Unlock()
	if ok {
		return strconv.FormatInt(id, 10), nil
	}
	b, _, err := c.do(http.MethodGet, "/api/orgs/lookup/"+url.PathEscape(name), nil)
	if err != nil {
		org, lerr := c.lookupOrgList(name)
		if lerr != nil {
			return "", err
		}
		c.rememberID("org:"+name, org.ID)
		return strconv.FormatInt(org.ID, 10), nil
	}
	var org Org
	if err := json.Unmarshal(b, &org); err != nil {
		return "", err
	}
	if org.ID == 0 {
		return "", fmt.Errorf("woodpecker org %s not found", name)
	}
	c.rememberID("org:"+name, org.ID)
	return strconv.FormatInt(org.ID, 10), nil
}

func (c *Client) lookupOrgList(name string) (Org, error) {
	b, _, err := c.do(http.MethodGet, "/api/user/orgs?page=1&perPage=50", nil)
	if err != nil {
		return Org{}, err
	}
	var orgs []Org
	if err := json.Unmarshal(b, &orgs); err != nil {
		return Org{}, err
	}
	for _, org := range orgs {
		if strings.EqualFold(org.Name, name) {
			return org, nil
		}
	}
	return Org{}, fmt.Errorf("woodpecker org %s not found", name)
}

func decodeSecrets(b []byte) []Secret {
	var raw []kernelSecret
	if err := json.Unmarshal(b, &raw); err != nil {
		return []Secret{}
	}
	out := make([]Secret, 0, len(raw))
	for _, s := range raw {
		out = append(out, publicSecret(s))
	}
	return out
}

func (c *Client) ListOrgSecrets(org string, q page.Query) (page.Result[Secret], error) {
	key, err := c.orgKey(org)
	if err != nil {
		return page.Result[Secret]{}, err
	}
	b, _, err := c.do(http.MethodGet, "/api/orgs/"+key+"/secrets", nil)
	if err != nil {
		return page.Result[Secret]{}, err
	}
	return page.Clip(decodeSecrets(b), q), nil
}

func (c *Client) ListRepoSecrets(fullName string, q page.Query) (page.Result[Secret], error) {
	key, err := c.repoKey(fullName)
	if err != nil {
		return page.Result[Secret]{}, err
	}
	b, _, err := c.do(http.MethodGet, "/api/repos/"+key+"/secrets", nil)
	if err != nil {
		return page.Result[Secret]{}, err
	}
	return page.Clip(decodeSecrets(b), q), nil
}

func (c *Client) hasSecret(list []Secret, name string) bool {
	for _, s := range list {
		if s.Name == name {
			return true
		}
	}
	return false
}

func (c *Client) PutOrgSecret(org, name, value string, events []string) (Secret, error) {
	key, err := c.orgKey(org)
	if err != nil {
		return Secret{}, err
	}
	listed, err := c.ListOrgSecrets(org, page.Query{Page: 1, Size: page.MaxSize})
	if err != nil {
		return Secret{}, err
	}
	return c.putSecret("/api/orgs/"+key+"/secrets", name, value, events, c.hasSecret(listed.Items, name))
}

func (c *Client) PutRepoSecret(fullName, name, value string, events []string) (Secret, error) {
	key, err := c.repoKey(fullName)
	if err != nil {
		return Secret{}, err
	}
	listed, err := c.ListRepoSecrets(fullName, page.Query{Page: 1, Size: page.MaxSize})
	if err != nil {
		return Secret{}, err
	}
	return c.putSecret("/api/repos/"+key+"/secrets", name, value, events, c.hasSecret(listed.Items, name))
}

func (c *Client) putSecret(base, name, value string, events []string, exists bool) (Secret, error) {
	body := secretBody(name, value, events)
	var (
		b   []byte
		err error
	)
	if exists {
		b, _, err = c.do(http.MethodPatch, base+"/"+url.PathEscape(name), body)
	} else {
		b, _, err = c.do(http.MethodPost, base, body)
	}
	if err != nil {
		return Secret{}, err
	}
	var raw kernelSecret
	if err := json.Unmarshal(b, &raw); err != nil {
		return Secret{Name: name, Events: events}, nil
	}
	out := publicSecret(raw)
	if out.Name == "" {
		out.Name = name
	}
	if len(out.Events) == 0 {
		out.Events = body["events"].([]string)
	}
	return out, nil
}

func (c *Client) DeleteOrgSecret(org, name string) error {
	key, err := c.orgKey(org)
	if err != nil {
		return err
	}
	_, _, err = c.do(http.MethodDelete, "/api/orgs/"+key+"/secrets/"+url.PathEscape(name), nil)
	return err
}

func (c *Client) DeleteRepoSecret(fullName, name string) error {
	key, err := c.repoKey(fullName)
	if err != nil {
		return err
	}
	_, _, err = c.do(http.MethodDelete, "/api/repos/"+key+"/secrets/"+url.PathEscape(name), nil)
	return err
}
