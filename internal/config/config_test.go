package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("UNIFI_URL", "https://10.0.0.1")
	t.Setenv("UNIFI_USER", "unpoller")
	t.Setenv("UNIFI_PASS", "secret")

	c, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Site != "default" {
		t.Errorf("Site = %q, want default", c.Site)
	}
	if c.Insecure {
		t.Errorf("Insecure = true, want false by default")
	}
	if c.PollInterval != 30*time.Second {
		t.Errorf("PollInterval = %s, want 30s", c.PollInterval)
	}
	if c.ListenAddr != ":8080" {
		t.Errorf("ListenAddr = %q, want :8080", c.ListenAddr)
	}
}

func TestLoadMissingRequired(t *testing.T) {
	t.Setenv("UNIFI_URL", "")
	t.Setenv("UNIFI_USER", "")
	t.Setenv("UNIFI_PASS", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for missing required vars")
	}
}

func TestLoadBadInterval(t *testing.T) {
	t.Setenv("UNIFI_URL", "https://10.0.0.1")
	t.Setenv("UNIFI_USER", "u")
	t.Setenv("UNIFI_PASS", "p")
	t.Setenv("POLL_INTERVAL", "not-a-duration")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for bad POLL_INTERVAL")
	}
}

func TestLoadInsecureParsing(t *testing.T) {
	t.Setenv("UNIFI_URL", "https://10.0.0.1")
	t.Setenv("UNIFI_USER", "u")
	t.Setenv("UNIFI_PASS", "p")
	cases := []struct {
		val  string
		want bool
	}{
		{"true", true},
		{"TRUE", true},
		{"1", true},
		{"false", false},
		{"0", false},
	}
	for _, tc := range cases {
		t.Setenv("UNIFI_INSECURE", tc.val)
		c, err := Load()
		if err != nil {
			t.Fatalf("Load with UNIFI_INSECURE=%q: %v", tc.val, err)
		}
		if c.Insecure != tc.want {
			t.Errorf("Insecure with %q = %v, want %v", tc.val, c.Insecure, tc.want)
		}
	}
}

func TestLoadInsecureInvalid(t *testing.T) {
	t.Setenv("UNIFI_URL", "https://10.0.0.1")
	t.Setenv("UNIFI_USER", "u")
	t.Setenv("UNIFI_PASS", "p")
	t.Setenv("UNIFI_INSECURE", "ture")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for unparseable UNIFI_INSECURE")
	}
}

func TestScrapeTimeout(t *testing.T) {
	cases := []struct {
		interval, want time.Duration
	}{
		{30 * time.Second, 30 * time.Second},
		{10 * time.Second, 10 * time.Second},
		{5 * time.Minute, 30 * time.Second},
	}
	for _, tc := range cases {
		c := Config{PollInterval: tc.interval}
		if got := c.ScrapeTimeout(); got != tc.want {
			t.Errorf("ScrapeTimeout with interval %s = %s, want %s", tc.interval, got, tc.want)
		}
	}
}

func TestLoadNonPositiveInterval(t *testing.T) {
	t.Setenv("UNIFI_URL", "https://10.0.0.1")
	t.Setenv("UNIFI_USER", "u")
	t.Setenv("UNIFI_PASS", "p")
	for _, v := range []string{"0s", "-5s"} {
		t.Setenv("POLL_INTERVAL", v)
		if _, err := Load(); err == nil {
			t.Fatalf("expected error for non-positive POLL_INTERVAL %q", v)
		}
	}
}
