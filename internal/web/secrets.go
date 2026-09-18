package web

import (
	"encoding/json"
	"net/http"

	"acahti/internal/page"
)

func (p *Pages) OrgSecrets(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	name := r.PathValue("name")
	switch r.Method {
	case http.MethodGet:
		out, err := p.cat(r).ListOrgSecrets(user, page.Parse(r))
		if err != nil {
			writeCatErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	case http.MethodPut:
		value, err := readSecretBody(r)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		out, err := p.cat(r).PutOrgSecret(user, name, value)
		if err != nil {
			writeCatErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	case http.MethodDelete:
		if err := p.cat(r).DeleteOrgSecret(user, name); err != nil {
			writeCatErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		writeErr(w, http.StatusMethodNotAllowed, "method")
	}
}

func (p *Pages) RepoSecrets(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	owner, name, secret := r.PathValue("owner"), r.PathValue("name"), r.PathValue("secret")
	switch r.Method {
	case http.MethodGet:
		out, err := p.Cat.ListRepoSecrets(user, owner, name, page.Parse(r))
		if err != nil {
			writeCatErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	case http.MethodPut:
		value, err := readSecretBody(r)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		out, err := p.Cat.PutRepoSecret(user, owner, name, secret, value)
		if err != nil {
			writeCatErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	case http.MethodDelete:
		if err := p.Cat.DeleteRepoSecret(user, owner, name, secret); err != nil {
			writeCatErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		writeErr(w, http.StatusMethodNotAllowed, "method")
	}
}

func readSecretBody(r *http.Request) (string, error) {
	var body struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return "", err
	}
	return body.Value, nil
}
