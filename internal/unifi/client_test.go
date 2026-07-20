package unifi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
