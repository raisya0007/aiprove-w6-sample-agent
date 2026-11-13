package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is a minimal AIProve HTTP client. It uses a browser User-Agent on
// every request (Cloudflare blocks default Go user agents) and attaches the
// Bearer token only on authenticated calls. It never logs the token.
type Client struct {
	base  string
	token string
	http  *http.Client
}

func NewClient(base, token string) *Client {
	return &Client{
		base:  strings.TrimRight(base, "/"),
		token: token,
		http:  &http.Client{Timeout: 45 * time.Second},
	}
}

// get performs an authenticated GET and decodes JSON into out (nil = raw body).
func (c *Client) get(path string, out any) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, c.base+path, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Authorization", "Bearer <redacted>")
	// Set the real token after building the log-safe header view.
	req.Header.Set("Authorization", "Bearer "+c.token)
	return c.do(req, out)
}

// getPublic performs an unauthenticated GET (skill.md, health).
func (c *Client) getPublic(path string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, c.base+path, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", UserAgent)
	return c.do(req, nil)
}

// postJSON performs an authenticated POST with a JSON body.
func (c *Client) postJSON(path string, body any, out any) (int, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequest(http.MethodPost, c.base+path, bytes.NewReader(raw))
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	_, status, err := c.do(req, out)
	return status, err
}

func (c *Client) do(req *http.Request, out any) ([]byte, int, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%s %s: %w", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return body, resp.StatusCode, fmt.Errorf("%s %s: HTTP %d: %s",
			req.Method, req.URL.Path, resp.StatusCode, truncate(string(body), 300))
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return body, resp.StatusCode, fmt.Errorf("decode %s: %w", req.URL.Path, err)
		}
	}
	return body, resp.StatusCode, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// --- API shapes (only the fields this agent uses) ---

type MeResponse struct {
	Address string `json:"address"`
	Profile struct {
		Handle string `json:"handle"`
	} `json:"profile"`
	Referral struct {
		Code string `json:"code"`
		Link string `json:"link"`
	} `json:"referral"`
	Competing []Bounty `json:"competing"`
}

type Bounty struct {
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Cat     string `json:"cat"`
	Lane    string `json:"lane"`
	Amt     string `json:"amt"`
	Unit    string `json:"unit"`
	Status  string `json:"status"`
	EndDate string `json:"endDate"`
	Brief   string `json:"brief"`
	Subs    int    `json:"subs"`
}

type BountyList struct {
	Bounties []Bounty `json:"bounties"`
	Count    int      `json:"count"`
}

type Submission struct {
	ID         string          `json:"id"`
	BountySlug string          `json:"bountySlug"`
	Prediction json.RawMessage `json:"prediction"`
	Status     string          `json:"status"`
}

type SubmissionsResponse struct {
	Submissions []Submission `json:"submissions"`
}

type SubmitResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}
