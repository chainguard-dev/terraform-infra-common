package httpmetrics

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chainguard-dev/clog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/shirou/gopsutil/v4/disk"
)

const (
	DiskUsageScrapeInterval    = 5 * time.Second
	DiskUsageScrapeIntervalEnv = "DISK_USAGE_SCRAPE_INTERVAL"
)

var (
	// Prometheus metrics for disk usage.
	diskUsageBytesGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "disk_usage_bytes",
			Help: "Disk usage in bytes.",
		},
		[]string{"mount"},
	)

	diskUsageScrapeFailures = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "disk_usage_scrape_failures",
			Help: "The number of failures when scraping disk usage.",
		},
	)
	// So that we only start the disk usage scraper once.
	once = new(sync.Once)
)

// scrapeDiskUsage returns the disk usage of all mounted partitions.
//
// On Cloud Run, there are 4 partitions:
// - /tmp
// - /
// - /dev
// - /dev/shm
// - and any additional volume mounts.
//
// We ignore all the /dev partitions here, keeping /, /tmp, and any additional
// volumn mounts that may come from the Revision config.
func scrapeDiskUsage() map[string]uint64 {
	parts, err := disk.Partitions(true)
	if err != nil {
		// It is better to be silent here and missing metrics, than to be spam log
		// here, and/or panic.
		diskUsageScrapeFailures.Inc()
		return nil
	}
	usage := make(map[string]uint64, len(parts))
	for _, p := range parts {
		device := p.Mountpoint
		s, err := disk.Usage(device)
		if err != nil || s == nil || s.Total == 0 {
			// Some Cloud Run partitions don't implement usage stats.
			continue
		}
		if strings.HasPrefix(device, "/dev") {
			// Ignore /dev partitions, nothing useful there.
			continue
		}
		usage[device] = s.Used
	}
	return usage
}

func ScrapeDiskUsage(ctx context.Context) {
	once.Do(func() {
		clog.FromContext(ctx).Info("Starting disk usage scraper with interval", "interval", DiskUsageScrapeInterval)
		// Check the env var for the interval.
		interval := scrapeInterval()
		// Start a timer to scrape disk usage every 5 seconds.
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				usage := scrapeDiskUsage()
				// report metrics
				for mount, used := range usage {
					diskUsageBytesGauge.WithLabelValues(mount).Set(float64(used))
				}
			case <-ctx.Done():
				return
			}
		}
	})
}

func scrapeInterval() time.Duration {
	interval := DiskUsageScrapeInterval

	if s, ok := os.LookupEnv(DiskUsageScrapeIntervalEnv); ok {
		if i, err := time.ParseDuration(s); err == nil {
			interval = i
		}
	}
	return interval
}
