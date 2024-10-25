package httpmetrics

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/chainguard-dev/clog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/shirou/gopsutil/disk"
)

const (
	DiskUsageScrapeInterval = 5 * time.Second
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

		// Start a timer to scrape disk usage every 5 seconds.
		ticker := time.NewTicker(DiskUsageScrapeInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				usage := scrapeDiskUsage()
				// report metrics
				for mount, used := range usage {
					diskUsageBytesGauge.WithLabelValues(mount).Set(float64(used))
					clog.FromContext(ctx).Info("Disk usage reported", "mount", mount, "used", used)
				}
			case <-ctx.Done():
				return
			}
		}
	})
}
