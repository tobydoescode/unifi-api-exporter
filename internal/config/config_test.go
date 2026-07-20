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
	if !c.Insecure {
		t.Errorf("Insecure = false, want true")
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
