package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/theinventor/oopsie_exceptions-cli/internal/config"
	"github.com/theinventor/oopsie_exceptions-cli/internal/credstore"
)

const (
	EnvAPIKey  = "OOPSIE_API_KEY"
	EnvAPIURL  = "OOPSIE_API_URL"
	EnvProject = "OOPSIE_PROJECT"

	AuthScheme      = "Bearer"
	UserAgentPrefix = "oopsie-cli"
)

type Client struct {
	BaseURL     string
	APIKey      string
	HTTPClient  *http.Client
	Version     string
	Source      string
	Backend     string
	ProfileName string
	ProjectID   string
	ProjectName string
}

func New() *Client {
	return NewWithOptions("", "")
}

func NewWithOptions(profile, apiURLOverride string) *Client {
	c := &Client{HTTPClient: &http.Client{Timeout: 30 * time.Second}}

	if profile != "" {
		if f, err := config.Load(); err == nil {
			if p, ok := f.Get(profile); ok {
				c.loadFromProfile(profile, p)
			}
		}
		c.applyURLFallbacks(apiURLOverride)
		return c
	}

	if envKey := os.Getenv(EnvAPIKey); envKey != "" {
		c.APIKey = envKey
		c.Source = "env"
		c.Backend = credstore.BackendEnv
		c.ProjectName = os.Getenv(EnvProject)
		c.applyURLFallbacks(apiURLOverride)
		return c
	}

	if f, err := config.Load(); err == nil {
		if p, ok := f.Get(""); ok {
			c.loadFromProfile(f.DefaultProfile, p)
		}
	}
	c.applyURLFallbacks(apiURLOverride)
	return c
}

func (c *Client) loadFromProfile(name string, p *config.Profile) {
	c.BaseURL = strings.TrimRight(p.APIURL, "/")
	c.Source = "profile:" + name
	c.ProfileName = name
	c.Backend = p.Backend
	if c.Backend == "" {
		c.Backend = credstore.BackendFile
	}
	c.ProjectID = p.ProjectID
	c.ProjectName = p.ProjectName
	if secret, err := credstore.Get(name, p.Backend, p.APIKey); err == nil {
		c.APIKey = secret
	}
}

func (c *Client) applyURLFallbacks(apiURLOverride string) {
	if envURL := os.Getenv(EnvAPIURL); envURL != "" && c.BaseURL == "" {
		c.BaseURL = envURL
	}
	if apiURLOverride != "" {
		c.BaseURL = apiURLOverride
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
}

func (c *Client) ReadyForRequest() error {
	if c.BaseURL == "" {
		return fmt.Errorf("missing Oopsie API URL; set --api-url, %s, or save a profile with `oopsie auth save --api-url <url>`", EnvAPIURL)
	}
	if c.APIKey == "" {
		return fmt.Errorf("missing Oopsie API key; set %s or run `oopsie auth save --profile <name> --api-key <key> --api-url <url>`", EnvAPIKey)
	}
	return nil
}

func (c *Client) MaskedAPIKey() string {
	return MaskKey(c.APIKey)
}

func MaskKey(key string) string {
	if key == "" {
		return "(none)"
	}
	if len(key) < 12 {
		return "***"
	}
	return key[:8] + "..." + key[len(key)-4:]
}

func (c *Client) Do(method, path string, body any, query url.Values) (*http.Response, error) {
	return c.DoWithHeaders(method, path, body, query, nil)
}

func (c *Client) DoWithHeaders(method, path string, body any, query url.Values, extra map[string]string) (*http.Response, error) {
	if err := c.ReadyForRequest(); err != nil {
		return nil, err
	}
	u := c.BaseURL + path
	if query != nil && len(query) > 0 {
		u += "?" + query.Encode()
	}
	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgentPrefix+"/"+c.Version)
	req.Header.Set("Authorization", AuthScheme+" "+c.APIKey)
	for k, v := range extra {
		if v != "" {
			req.Header.Set(k, v)
		}
	}
	return c.HTTPClient.Do(req)
}
