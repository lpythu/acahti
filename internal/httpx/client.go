package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const DefaultTimeout = 45 * time.Second

type Client struct {
	Base string
	HTTP *http.Client
	// Timeout returns a per-request deadline. Zero keeps the HTTP client timeout only.
	Timeout func(method, path string) time.Duration
	// OnError maps a >=400 response to an error. If nil, a generic status error is used.
	OnError func(method, path string, status int, body []byte) error
}

type Result struct {
	Status int
	Body   []byte
	Header http.Header
}

type Option func(*options)

type options struct {
	softFail bool
}

// SoftFail returns the response for >=400 without treating it as an error.
func SoftFail() Option {
	return func(o *options) { o.softFail = true }
}

func New(base string) *Client {
	return &Client{
		Base: strings.TrimRight(base, "/"),
		HTTP: &http.Client{Timeout: DefaultTimeout},
	}
}

func (c *Client) timeout(method, path string) time.Duration {
	if c != nil && c.Timeout != nil {
		return c.Timeout(method, path)
	}
	return 0
}

func (c *Client) Do(ctx context.Context, method, path string, body any, hdr http.Header, opts ...Option) (Result, error) {
	if c == nil {
		return Result{}, fmt.Errorf("httpx: nil client")
	}
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if d := c.timeout(method, path); d > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d)
		defer cancel()
	}
	var rdr io.Reader
	switch b := body.(type) {
	case nil:
	case []byte:
		rdr = bytes.NewReader(b)
	case string:
		rdr = strings.NewReader(b)
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			return Result{}, err
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Base+path, rdr)
	if err != nil {
		return Result{}, err
	}
	for k, vs := range hdr {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	if body != nil && req.Header.Get("Content-Type") == "" {
		switch body.(type) {
		case []byte, string:
		default:
			req.Header.Set("Content-Type", "application/json")
		}
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultTimeout}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	out := Result{Status: resp.StatusCode, Body: raw, Header: resp.Header.Clone()}
	if resp.StatusCode >= 400 && !o.softFail {
		if c.OnError != nil {
			return out, c.OnError(method, path, resp.StatusCode, raw)
		}
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return out, fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode, msg)
	}
	return out, nil
}

func (c *Client) JSON(ctx context.Context, method, path string, body, dest any, hdr http.Header, opts ...Option) (Result, error) {
	res, err := c.Do(ctx, method, path, body, hdr, opts...)
	if err != nil {
		return res, err
	}
	if dest == nil || len(res.Body) == 0 {
		return res, nil
	}
	if err := json.Unmarshal(res.Body, dest); err != nil {
		return res, err
	}
	return res, nil
}
