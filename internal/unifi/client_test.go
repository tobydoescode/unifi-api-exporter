package unifi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeController serves login + stat/device, requiring a cookie set by login.
func fakeController(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "TOKEN", Value: "abc", Path: "/"})
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/proxy/network/api/s/default/stat/device", func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie("TOKEN"); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[
			{"name":"Office AP","mac":"aa","type":"uap","model":"U7NHD","state":1},
			{"name":"Bedroom 2 AP","mac":"bb","type":"uap","model":"UAL6","state":10}
		]}`))
	})
	return httptest.NewTLSServer(mux)
}

func TestDevicesLoginAndParse(t *testing.T) {
	srv := fakeController(t)
	defer srv.Close()

	c, err := New(srv.URL, "u", "p", "default", true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	devs, err := c.Devices(context.Background())
	if err != nil {
		t.Fatalf("Devices: %v", err)
	}
	if len(devs) != 2 {
		t.Fatalf("got %d devices, want 2", len(devs))
	}
	if devs[1].Name != "Bedroom 2 AP" || devs[1].State != 10 {
		t.Errorf("device[1] = %+v, want Bedroom 2 AP state 10", devs[1])
	}
}

func TestLoginFailureBacksOff(t *testing.T) {
	var loginHits int
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, _ *http.Request) {
		loginHits++
		w.WriteHeader(http.StatusForbidden)
	})
	mux.HandleFunc("/proxy/network/api/s/default/stat/device", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c, err := New(srv.URL, "u", "wrong", "default", true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.Devices(context.Background()); err == nil {
		t.Fatal("expected error from failed login")
	}
	if loginHits != 1 {
		t.Fatalf("loginHits = %d, want 1", loginHits)
	}
	// Second poll inside the cooldown must not hit the login endpoint again.
	if _, err := c.Devices(context.Background()); err == nil {
		t.Fatal("expected error while backed off")
	}
	if loginHits != 1 {
		t.Errorf("loginHits = %d after backed-off poll, want 1", loginHits)
	}
}

func TestLoginBackoffGrowsAndResets(t *testing.T) {
	var loginHits int
	fail := true
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, _ *http.Request) {
		loginHits++
		if fail {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "TOKEN", Value: "abc", Path: "/"})
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/proxy/network/api/s/default/stat/device", func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie("TOKEN"); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c, err := New(srv.URL, "u", "p", "default", true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for i := 1; i <= 2; i++ {
		if _, derr := c.Devices(context.Background()); derr == nil {
			t.Fatalf("poll %d: expected error", i)
		}
		c.nextLoginTry = time.Time{} // simulate cooldown elapsing
	}
	if loginHits != 2 {
		t.Fatalf("loginHits = %d, want 2", loginHits)
	}
	if c.loginFailures != 2 {
		t.Errorf("loginFailures = %d, want 2", c.loginFailures)
	}
	fail = false
	if _, err := c.Devices(context.Background()); err != nil {
		t.Fatalf("Devices after login recovers: %v", err)
	}
	if c.loginFailures != 0 {
		t.Errorf("loginFailures = %d after success, want 0", c.loginFailures)
	}
}

func TestDevicesPersistentUnauthorized(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/proxy/network/api/s/default/stat/device", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c, err := New(srv.URL, "u", "p", "default", true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.Devices(context.Background()); err == nil {
		t.Fatal("expected error when still 401 after re-login")
	}
}

func TestDevicesBadJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/proxy/network/api/s/default/stat/device", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data": not-json`))
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c, err := New(srv.URL, "u", "p", "default", true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.Devices(context.Background()); err == nil {
		t.Fatal("expected error for malformed device JSON")
	}
}
