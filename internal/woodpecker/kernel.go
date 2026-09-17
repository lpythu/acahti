package woodpecker

import "encoding/json"

// DecodeKernel maps Woodpecker JSON (workflows / children) into Acahti
// pipeline → jobs → steps. Call only on kernel bytes.
func DecodeKernel(data []byte) (Pipeline, error) {
	var k struct {
		ID        int64       `json:"id"`
		Number    int64       `json:"number"`
		Status    string      `json:"status"`
		Event     string      `json:"event"`
		Branch    string      `json:"branch"`
		Ref       string      `json:"ref"`
		Title     string      `json:"title"`
		Message   string      `json:"message"`
		Author    string      `json:"author"`
		Avatar    string      `json:"avatar"`
		Commit    string      `json:"commit"`
		Error     string      `json:"error"`
		Errors    []PipeError `json:"errors"`
		Created   int64       `json:"created"`
		Started   int64       `json:"started"`
		Finished  int64       `json:"finished"`
		Workflows []struct {
			ID        int64    `json:"id"`
			PID       int64    `json:"pid"`
			Name      string   `json:"name"`
			State     string   `json:"state"`
			DependsOn []string `json:"depends_on"`
			Children  []Step   `json:"children"`
		} `json:"workflows"`
	}
	if err := json.Unmarshal(data, &k); err != nil {
		return Pipeline{}, err
	}
	p := Pipeline{
		ID: k.ID, Number: k.Number, Status: k.Status, Event: k.Event,
		Branch: k.Branch, Ref: k.Ref, Title: k.Title, Message: k.Message,
		Author: k.Author, Avatar: k.Avatar, Commit: k.Commit, Error: k.Error,
		Errors: k.Errors, Created: k.Created, Started: k.Started, Finished: k.Finished,
	}
	for _, w := range k.Workflows {
		steps := w.Children
		if len(steps) == 0 && w.PID > 0 {
			steps = []Step{{PID: w.PID, Name: w.Name, State: w.State}}
		}
		p.Jobs = append(p.Jobs, Job{ID: w.ID, PID: w.PID, Name: w.Name, State: w.State, DependsOn: w.DependsOn, Steps: steps})
	}
	p.flattenError()
	p.SortJobs()
	return p, nil
}

func DecodeKernelList(data []byte) ([]Pipeline, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := make([]Pipeline, 0, len(raw))
	for _, item := range raw {
		p, err := DecodeKernel(item)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}
