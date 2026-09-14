package web

import (
	"encoding/json"
	"errors"
	"net/http"

	"acahti/internal/catalog"
)

func writeCatErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, catalog.ErrForbidden):
		writeErr(w, http.StatusForbidden, err.Error())
	case errors.Is(err, catalog.ErrNotFound):
		writeErr(w, http.StatusNotFound, err.Error())
	case errors.Is(err, catalog.ErrInvalid):
		writeErr(w, http.StatusBadRequest, err.Error())
	default:
		writeErr(w, http.StatusBadGateway, err.Error())
	}
}

func (p *Pages) Groups(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	name := r.PathValue("group")
	switch r.Method {
	case http.MethodPost:
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		out, err := p.Cat.CreateGroup(user, body.Name)
		if err != nil {
			writeCatErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	case http.MethodDelete:
		if err := p.Cat.DeleteGroup(user, name); err != nil {
			writeCatErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		out, err := p.Cat.GroupAccess(user, name)
		if err != nil {
			writeCatErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func (p *Pages) GroupMember(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	group, login := r.PathValue("group"), r.PathValue("login")
	switch r.Method {
	case http.MethodDelete:
		if err := p.Cat.RemoveGroupMember(user, group, login); err != nil {
			writeCatErr(w, err)
			return
		}
	default:
		var body struct {
			Permission string `json:"permission"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if err := p.Cat.SetGroupMember(user, group, login, body.Permission); err != nil {
			writeCatErr(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (p *Pages) GroupRepo(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	group, repo := r.PathValue("group"), r.PathValue("repo")
	switch r.Method {
	case http.MethodDelete:
		if err := p.Cat.RemoveGroupRepo(user, group, repo); err != nil {
			writeCatErr(w, err)
			return
		}
	default:
		if err := p.Cat.AddGroupRepo(user, group, repo); err != nil {
			writeCatErr(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (p *Pages) RepoAccess(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	out, err := p.Cat.RepoAccess(user, r.PathValue("owner"), r.PathValue("name"))
	if err != nil {
		writeCatErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (p *Pages) RepoCollaborator(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	owner, name, login := r.PathValue("owner"), r.PathValue("name"), r.PathValue("login")
	switch r.Method {
	case http.MethodDelete:
		if err := p.Cat.RemoveCollaborator(user, owner, name, login); err != nil {
			writeCatErr(w, err)
			return
		}
	default:
		var body struct {
			Permission string `json:"permission"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if err := p.Cat.SetCollaborator(user, owner, name, login, body.Permission); err != nil {
			writeCatErr(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
