package catalog

import "time"

const heatWeeks = 53

type HeatDay struct {
	Date  string `json:"date"`
	Value int64  `json:"value"`
}

type Heatmap struct {
	Total int64     `json:"total"`
	Days  []HeatDay `json:"days"`
}

func (c *Catalog) BoardHeatmap(user string) (Heatmap, error) {
	now := time.Now()
	start, end := heatRange(now)
	if c.Idx == nil {
		return fillHeatmap(nil, start, end, now), nil
	}
	names, err := c.visiblePipeRepos(user, "")
	if err != nil {
		return Heatmap{}, err
	}
	counts, err := c.Idx.DailyCounts(names, start.Unix(), end.Unix())
	if err != nil {
		return Heatmap{}, err
	}
	return fillHeatmap(counts, start, end, now), nil
}

func heatRange(now time.Time) (start, end time.Time) {
	now = now.UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start = today.AddDate(0, 0, -364)
	for start.Weekday() != time.Monday {
		start = start.AddDate(0, 0, -1)
	}
	return start, start.AddDate(0, 0, heatWeeks*7)
}

func fillHeatmap(counts map[string]int64, start, end, now time.Time) Heatmap {
	today := now.UTC()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	out := Heatmap{Days: make([]HeatDay, 0, heatWeeks*7)}
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		n := counts[key]
		out.Days = append(out.Days, HeatDay{Date: key, Value: n})
		if !d.After(today) {
			out.Total += n
		}
	}
	return out
}
