package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// client calls Kayra: url its base, the groups' roots from its
// /.well-known/kayra.json.
type client struct {
	url       string
	token     string
	userAgent string
	http      *http.Client
	roots     map[string]string // system, catalog, authorization, events: /apis/<group>/v1
}

// errNotFound is a 404: the object is gone.
var errNotFound = errors.New("not found")

// problem is Kayra's error, a problem document (RFC 9457).
type problem struct {
	Status   int    `json:"status"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Solution string `json:"solution"`
}

// apiError is an answer of Kayra's other than a success.
type apiError struct {
	Method, Path string
	Problem      problem
}

func (e *apiError) Error() string {
	msg := fmt.Sprintf("%s %s: %d", e.Method, e.Path, e.Problem.Status)
	if e.Problem.Title != "" {
		msg += " " + e.Problem.Title
	}
	if e.Problem.Detail != "" {
		msg += ": " + e.Problem.Detail
	}
	if e.Problem.Solution != "" {
		msg += " (" + e.Problem.Solution + ")"
	}
	return msg
}

func (e *apiError) Is(target error) bool {
	return target == errNotFound && e.Problem.Status == http.StatusNotFound
}

// newClient discovers Kayra's groups at base; its requests carry userAgent.
func newClient(ctx context.Context, base, token, userAgent string) (*client, error) {
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("url %q: Kayra's, like https://kayra.example.com", base)
	}
	c := &client{url: strings.TrimRight(base, "/"), token: token, userAgent: userAgent, http: &http.Client{Timeout: 60 * time.Second}}
	var doc struct {
		APIs map[string]string `json:"apis"`
	}
	if err := c.do(ctx, http.MethodGet, "/.well-known/kayra.json", nil, &doc); err != nil {
		return nil, fmt.Errorf("discovering Kayra at %s: %w", c.url, err)
	}
	for _, api := range []string{"system", "catalog", "authorization"} {
		if doc.APIs[api] == "" {
			return nil, fmt.Errorf("%s/.well-known/kayra.json names no %s API: is it Kayra?", c.url, api)
		}
	}
	c.roots = doc.APIs
	return c, nil
}

// path is the path of a resource under api's root, its {attr} replaced by
// the values in attrs, each escaped.
func (c *client) path(api, tmpl string, attrs map[string]string) (string, error) {
	root, ok := c.roots[api]
	if !ok {
		return "", fmt.Errorf("no %s API in Kayra's /.well-known/kayra.json", api)
	}
	var missing error
	p := regexp.MustCompile(`\{([^}]+)\}`).ReplaceAllStringFunc(tmpl, func(m string) string {
		v := attrs[m[1:len(m)-1]]
		if v == "" {
			missing = fmt.Errorf("%s: %s is not known", tmpl, m)
		}
		return url.PathEscape(v)
	})
	return root + p, missing
}

// do sends a request to path, in is its JSON body, out decodes the
// answer's.
func (c *client) do(ctx context.Context, method, path string, in, out any) error {
	var r io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		r = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.url+path, r) //#nosec G704 -- Kayra's URL, the provider's configuration; path escaped
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req) //#nosec G704 -- Kayra's URL, the provider's configuration
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		e := &apiError{Method: method, Path: path, Problem: problem{Status: resp.StatusCode}}
		if json.Unmarshal(data, &e.Problem) != nil || e.Problem.Title == "" && e.Problem.Detail == "" {
			e.Problem.Status, e.Problem.Title = resp.StatusCode, http.StatusText(resp.StatusCode)
			e.Problem.Detail = strings.TrimSpace(string(data))
		}
		if e.Problem.Status == 0 {
			e.Problem.Status = resp.StatusCode
		}
		return e
	}
	if out == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("%s %s: decoding the answer: %w", method, path, err)
	}
	return nil
}
