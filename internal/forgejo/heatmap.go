package forgejo

import (
	"encoding/json"
	"net/http"
	"net/url"
)

type HeatPoint struct {
	Timestamp     int64 `json:"timestamp"`
	Contributions int64 `json:"contributions"`
}

func (c *Client) UserHeatmap(login string) ([]HeatPoint, error) {
	if !c.Ready() || login == "" {
		return nil, nil
	}
	b, _, err := c.do(http.MethodGet, "/api/v1/users/"+url.PathEscape(login)+"/heatmap", "", login, nil)
	if err != nil {
		return nil, err
	}
	var out []HeatPoint
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}
