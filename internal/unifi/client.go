package unifi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

// Device is the subset of a UniFi controller device we export.
type Device struct {
	Name  string `json:"name"`
	Mac   string `json:"mac"`
	Type  string `json:"type"`
	Model string `json:"model"`
	State int    `json:"state"`
}

// Login backoff bounds: repeated failed logins can trip UniFi OS rate
// limiting and lock the account, so we cool down between attempts.
const (
	loginBackoffBase = time.Minute
	loginBackoffMax  = 15 * time.Minute
)

// Client talks to a UniFi OS controller (UDM-style /proxy/network paths).
// Not safe for concurrent use; the poll loop is the only caller.
type Client struct {
	baseURL string
	user    string
	pass    string
	site    string
	http    *http.Client

	loginFailures int
	nextLoginTry  time.Time
}

// New builds a Client with a cookie jar; insecure skips TLS verification.
// Request timeouts are the caller's job via context.
func New(baseURL, user, pass, site string, insecure bool) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		user:    user,
		pass:    pass,
		site:    site,
		http: &http.Client{
			Jar: jar,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}, //nolint:gosec // self-signed UDM cert
			},
		},
	}, nil
}

// Devices fetches the device list, logging in and retrying once on 401.
func (c *Client) Devices(ctx context.Context) ([]Device, error) {
	devs, status, err := c.getDevices(ctx)
	if err == nil {
		return devs, nil
	}
	if status == http.StatusUnauthorized {
		if lerr := c.login(ctx); lerr != nil {
			return nil, lerr
		}
		devs, _, err = c.getDevices(ctx)
		return devs, err
	}
	return nil, err
}

func (c *Client) login(ctx context.Context) error {
	if now := time.Now(); now.Before(c.nextLoginTry) {
		return fmt.Errorf("login backed off until %s after %d consecutive failures",
			c.nextLoginTry.Format(time.RFC3339), c.loginFailures)
	}
	if err := c.doLogin(ctx); err != nil {
		c.loginFailures++
		backoff := loginBackoffMax
		if shift := c.loginFailures - 1; shift < 8 && loginBackoffBase<<shift < loginBackoffMax {
			backoff = loginBackoffBase << shift
		}
		c.nextLoginTry = time.Now().Add(backoff)
		return err
	}
	c.loginFailures = 0
	c.nextLoginTry = time.Time{}
	return nil
}

func (c *Client) doLogin(ctx context.Context) error {
	body, err := json.Marshal(map[string]string{"username": c.user, "password": c.pass})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/auth/login", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) getDevices(ctx context.Context) ([]Device, int, error) {
	url := fmt.Sprintf("%s/proxy/network/api/s/%s/stat/device", c.baseURL, c.site)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("stat/device: status %d", resp.StatusCode)
	}
	var out struct {
		Data []Device `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, resp.StatusCode, err
	}
	return out.Data, resp.StatusCode, nil
}
