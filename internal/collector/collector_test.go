package collector

import (
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/tobydoescode/unifi-api-exporter/internal/unifi"
)

func TestObserveSuccess(t *testing.T) {
	reg := prometheus.NewRegistry()
	c := New(reg, "default")

	c.Observe([]unifi.Device{
		{Name: "Bedroom 2 AP", Mac: "bb", Type: "uap", Model: "UAL6", State: 10},
	}, 42*time.Millisecond, nil)

	got := testutil.ToFloat64(c.state.WithLabelValues("Bedroom 2 AP", "bb", "uap", "UAL6", "default"))
	if got != 10 {
		t.Errorf("state = %v, want 10", got)
	}
	if s := testutil.ToFloat64(c.success); s != 1 {
		t.Errorf("scrape_success = %v, want 1", s)
	}
}

func TestObserveErrorRetainsLastGood(t *testing.T) {
	reg := prometheus.NewRegistry()
	c := New(reg, "default")

	c.Observe([]unifi.Device{
		{Name: "Office AP", Mac: "aa", Type: "uap", Model: "U7NHD", State: 1},
	}, time.Millisecond, nil)
	c.Observe(nil, time.Millisecond, errors.New("boom"))

	if s := testutil.ToFloat64(c.success); s != 0 {
		t.Errorf("scrape_success = %v, want 0", s)
	}
	// Last-good series is retained (not reset) on failure.
	got := testutil.ToFloat64(c.state.WithLabelValues("Office AP", "aa", "uap", "U7NHD", "default"))
	if got != 1 {
		t.Errorf("retained state = %v, want 1", got)
	}
}
