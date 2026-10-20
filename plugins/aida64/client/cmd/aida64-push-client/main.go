package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"aida64-client/internal/collector"
	"aida64-client/internal/publisher"
)

type Config struct {
	Device       string
	Interval     time.Duration
	Timeout      time.Duration
	StorageUsage string
	StoragePath  string
	StorageDisk  string
	StorageTemp  string
	CPUName      string
	RAMSpeed     string
	Once         bool
	Token        string
	LHMURL       string
	LHMFile      string
}

func main() {
	cfg := parseFlags()
	if cfg.Interval <= 0 {
		log.Fatalf("invalid --interval: must be > 0")
	}

	if cfg.Token == "" {
		log.Fatalf("missing --token (or AIDA64_FACE_TOKEN): create a face push token on the Device Console, Display page")
	}
	pub := publisher.New(cfg.Device, cfg.Token, cfg.Timeout)
	col := collector.NewHostCollector(collector.HostCollectorConfig{
		StorageUsageValue: cfg.StorageUsage,
		StoragePath:       cfg.StoragePath,
		StorageDisk:       cfg.StorageDisk,
		StorageTemp:       cfg.StorageTemp,
		CPUName:           cfg.CPUName,
		RAMSpeed:          cfg.RAMSpeed,
		LHMURL:            cfg.LHMURL,
		LHMFile:           cfg.LHMFile,
	})

	ctx := context.Background()
	for {
		cycleStart := time.Now()

		metrics, err := col.Collect(ctx)
		if err != nil {
			log.Printf("[collect] failed: %v", err)
		} else {
			resp, err := pub.PushMetrics(ctx, metrics)
			if err != nil {
				log.Printf("[push] failed: %v", err)
			} else {
				log.Printf("[push] ok ram_speed=%q ram_usage=%s ram_free=%s ram_load=%s cpu_name=%q cpu_load=%s cpu_temp=%s cpu_clock=%s cpu_fan=%s storage_disk=%q storage_usage=%s storage_temp=%s storage_used_pct=%s resp=%s",
					metrics.RAMSpeed,
					metrics.RAMUsage,
					metrics.RAMFree,
					metrics.RAMLoad,
					metrics.CPUName,
					metrics.CPULoad,
					metrics.CPUTemp,
					metrics.CPUClock,
					metrics.CPUFan,
					metrics.StorageDisk,
					metrics.StorageUsage,
					metrics.StorageTemp,
					metrics.StorageUsedPct,
					resp)
			}
		}

		if cfg.Once {
			break
		}

		sleep := cfg.Interval - time.Since(cycleStart)
		if sleep > 0 {
			time.Sleep(sleep)
		}
	}
}

func parseFlags() Config {
	cfg := Config{}
	flag.StringVar(&cfg.Device, "device", "http://127.0.0.1", "Device base URL, e.g. http://192.168.1.50 (or include a non-default port)")
	flag.DurationVar(&cfg.Interval, "interval", time.Second, "Push interval (example: 1s, 500ms)")
	flag.DurationVar(&cfg.Timeout, "timeout", 3*time.Second, "HTTP timeout")
	flag.StringVar(&cfg.StorageUsage, "storage-usage", "", "Storage usage text override (optional)")
	flag.StringVar(&cfg.StoragePath, "storage-path", "", "Filesystem path used for storage usage (default: / or C:\\)")
	flag.StringVar(&cfg.StorageDisk, "storage-disk", "", "Storage disk name override (optional)")
	flag.StringVar(&cfg.StorageTemp, "storage-temp", "", "Storage temperature override (optional, default: N/A)")
	flag.StringVar(&cfg.CPUName, "cpu-name", "", "CPU name override (auto-detected if empty)")
	flag.StringVar(&cfg.RAMSpeed, "ram-speed", "", "RAM speed override (auto-detected best effort if empty)")
	flag.StringVar(&cfg.RAMSpeed, "ram-type", "", "Deprecated alias of --ram-speed")
	flag.BoolVar(&cfg.Once, "once", false, "Send one payload and exit")
	flag.StringVar(&cfg.Token, "token", os.Getenv("AIDA64_FACE_TOKEN"), "Face push token from the Device Console (default: $AIDA64_FACE_TOKEN)")
	flag.StringVar(&cfg.LHMURL, "lhm-url", "http://127.0.0.1:8085/data.json", "Windows LibreHardwareMonitor JSON URL")
	flag.StringVar(&cfg.LHMFile, "lhm-file", "", "Windows LibreHardwareMonitor JSON file path (optional)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "AIDA64 Push Client (Go)\n\n")
		fmt.Fprintf(os.Stderr, "Pushes metrics to the device's AIDA64 face (%s, with a face push token)\n\n", publisher.ValuesPath)
		flag.PrintDefaults()
	}
	flag.Parse()
	if cfg.StorageUsage == "" {
		cfg.StorageUsage = "N/A"
	}
	return cfg
}
