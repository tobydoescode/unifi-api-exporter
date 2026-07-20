package collector

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/tobydoescode/unifi-api-exporter/internal/unifi"
)

// Collector holds the exported UniFi metrics.
type Collector struct {
	site    string
	state   *prometheus.GaugeVec
	success prometheus.Gauge
	dur     prometheus.Gauge
}

// New registers the metrics and returns a Collector.
func New(reg prometheus.Registerer, site string) *Collector {
	c := &Collector{
		site: site,
		state: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "unifi_device_state",
			Help: "UniFi device state from the controller (1=connected/online).",
		}, []string{"name", "mac", "type", "model", "site"}),
		success: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "unifi_scrape_success",
			Help: "1 if the last controller poll succeeded, else 0.",
		}),
		dur: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "unifi_scrape_duration_seconds",
			Help: "Duration of the last controller poll in seconds.",
		}),
	}
	reg.MustRegister(c.state, c.success, c.dur)
	return c
}

// Observe records the result of one poll. On error it flips scrape_success to 0
// and leaves the last-good device gauges in place.
func (c *Collector) Observe(devs []unifi.Device, d time.Duration, scrapeErr error) {
	c.dur.Set(d.Seconds())
	if scrapeErr != nil {
		c.success.Set(0)
		return
	}
	c.success.Set(1)
	c.state.Reset()
	for _, dev := range devs {
		c.state.WithLabelValues(dev.Name, dev.Mac, dev.Type, dev.Model, c.site).Set(float64(dev.State))
	}
}
