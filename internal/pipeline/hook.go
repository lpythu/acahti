package pipeline

import (
	"encoding/json"
	"io"
	"net/http"
)

type fileMeta struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type configRequest struct {
	Configuration []fileMeta `json:"configuration"`
}

type configResponse struct {
	Configs []fileMeta `json:"configs"`
}

// HandleConfig expands pipe: in pipeline YAML. Auth is basic user "pipe" or query token=.
func HandleConfig(token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		q := r.URL.Query().Get("token")
		authed := (ok && user == "pipe" && pass == token) || q == token
		if token == "" || !authed {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var req configRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		out := make([]fileMeta, 0, len(req.Configuration))
		for _, cfg := range req.Configuration {
			data, err := Expand([]byte(cfg.Data))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			out = append(out, fileMeta{Name: cfg.Name, Data: string(data)})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(configResponse{Configs: out})
	}
}
