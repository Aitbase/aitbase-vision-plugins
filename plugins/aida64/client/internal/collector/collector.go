package collector

import (
	"context"
	"fmt"
	"math"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

// Metrics is the LCD-facing AIDA64 payload.
type Metrics struct {
	RAMSpeed     string `json:"ram_speed"`
	RAMUsage     string `json:"ram_usage"`
	RAMFree      string `json:"ram_free"`
	RAMLoad      string `json:"ram_load"`
	CPUName      string `json:"cpu_name"`
	CPULoad      string `json:"cpu_load"`
	CPUTemp      string `json:"cpu_temp"`
	CPUClock     string `json:"cpu_clock"`
	CPUFan       string `json:"cpu_fan"`
	StorageDisk  string `json:"storage_disk"`
	StorageUsage string `json:"storage_usage"`
	StorageTemp  string `json:"storage_temp"`
	// The disk's used share as a whole percent ("60"), for the face's disk bar; "" when unknown.
	StorageUsedPct string `json:"storage_used_pct"`
}

// Collector fetches host metrics and formats them for the API.
type Collector interface {
	Collect(ctx context.Context) (Metrics, error)
}

// TempProvider resolves a host temperature in Celsius.
type TempProvider interface {
	TemperatureC(ctx context.Context) (float64, error)
}

type fanProvider interface {
	FanRPM(ctx context.Context) (int, error)
}

type HostCollector struct {
	tempProvider TempProvider
	storagePath  string

	storageUsage string
	cpuName      string
	ramSpeed     string
	storageDisk  string
	storageTemp  string

	staticOnce sync.Once
	runtimeMu  sync.Mutex
	lastFanRPM int
}

type HostCollectorConfig struct {
	StorageUsageValue string
	StoragePath       string
	CPUName           string
	RAMSpeed          string
	StorageDisk       string
	StorageTemp       string

	// Windows-only source for LibreHardwareMonitor JSON.
	LHMURL  string
	LHMFile string
}

func NewHostCollector(cfg HostCollectorConfig) *HostCollector {
	usage := strings.TrimSpace(cfg.StorageUsageValue)
	if usage == "" {
		usage = "N/A"
	}

	c := &HostCollector{
		storagePath:  defaultStoragePath(cfg.StoragePath),
		storageUsage: usage,
		cpuName:      strings.TrimSpace(cfg.CPUName),
		ramSpeed:     strings.TrimSpace(cfg.RAMSpeed),
		storageDisk:  strings.TrimSpace(cfg.StorageDisk),
		storageTemp:  strings.TrimSpace(cfg.StorageTemp),
	}
	if c.storageTemp == "" {
		c.storageTemp = "N/A"
	}
	if runtime.GOOS == "linux" {
		c.tempProvider = NewLinuxHwmonProvider("/sys/class/hwmon")
	} else if runtime.GOOS == "windows" {
		c.tempProvider = NewWindowsLHMProvider(cfg.LHMURL, cfg.LHMFile)
	}
	return c
}

func (c *HostCollector) Collect(ctx context.Context) (Metrics, error) {
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return Metrics{}, fmt.Errorf("virtual memory: %w", err)
	}

	c.ensureStaticInfo(ctx)

	m := Metrics{
		RAMSpeed:     c.ramSpeed,
		RAMUsage:     formatBytes(vm.Used),
		RAMFree:      formatBytes(vm.Available),
		RAMLoad:      formatPercent(vm.UsedPercent),
		CPUName:      c.cpuName,
		CPULoad:      detectCPULoad(ctx),
		CPUTemp:      "N/A",
		CPUClock:     detectCPUClock(ctx),
		CPUFan:       "N/A",
		StorageDisk:  c.storageDisk,
		StorageUsage: c.storageUsage,
		StorageTemp:  c.storageTemp,
	}

	if c.tempProvider != nil {
		tempC, err := c.tempProvider.TemperatureC(ctx)
		if err == nil && !math.IsNaN(tempC) && !math.IsInf(tempC, 0) {
			m.CPUTemp = fmt.Sprintf("%.0fC", tempC)
		}
		if fp, ok := c.tempProvider.(fanProvider); ok {
			if fanRPM, fanErr := fp.FanRPM(ctx); fanErr == nil && fanRPM > 0 {
				c.runtimeMu.Lock()
				c.lastFanRPM = fanRPM
				c.runtimeMu.Unlock()
				m.CPUFan = fmt.Sprintf("%d RPM", fanRPM)
			} else {
				c.runtimeMu.Lock()
				cachedFan := c.lastFanRPM
				c.runtimeMu.Unlock()
				if cachedFan > 0 {
					m.CPUFan = fmt.Sprintf("%d RPM", cachedFan)
				}
			}
		}
	}

	if usage, err := disk.UsageWithContext(ctx, c.storagePath); err == nil && usage.Total > 0 {
		m.StorageUsage = fmt.Sprintf("%s/%s", formatBytes(usage.Used), formatBytes(usage.Total))
		m.StorageUsedPct = fmt.Sprintf("%d", int(usage.UsedPercent+0.5))
	}

	return m, nil
}

func (c *HostCollector) ensureStaticInfo(ctx context.Context) {
	c.staticOnce.Do(func() {
		if c.cpuName == "" {
			c.cpuName = detectCPUName(ctx)
		}
		if c.cpuName == "" {
			c.cpuName = "Unknown CPU"
		}

		if c.ramSpeed == "" {
			c.ramSpeed = detectRAMSpeed()
		}
		if c.ramSpeed == "" {
			c.ramSpeed = "N/A"
		}

		if c.storageDisk == "" {
			c.storageDisk = detectStorageDiskName(ctx, c.storagePath)
		}
		if c.storageDisk == "" {
			c.storageDisk = "Main Disk"
		}
	})
}

func detectCPUName(ctx context.Context) string {
	info, err := cpu.InfoWithContext(ctx)
	if err != nil || len(info) == 0 {
		return ""
	}
	model := strings.TrimSpace(info[0].ModelName)
	if model == "" {
		return ""
	}

	physicalCount, physicalErr := cpu.CountsWithContext(ctx, false)
	if physicalErr == nil && physicalCount > 0 {
		return fmt.Sprintf("%s x %d", model, physicalCount)
	}

	logicalCount, logicalErr := cpu.CountsWithContext(ctx, true)
	if logicalErr == nil && logicalCount > 0 {
		return fmt.Sprintf("%s x %d", model, logicalCount)
	}
	return model
}

func detectCPUClock(ctx context.Context) string {
	info, err := cpu.InfoWithContext(ctx)
	if err != nil || len(info) == 0 {
		return "N/A"
	}
	mhz := info[0].Mhz
	if mhz <= 0 {
		return "N/A"
	}
	return fmt.Sprintf("%.0f MHz", mhz)
}

func detectCPULoad(ctx context.Context) string {
	percents, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil || len(percents) == 0 {
		return "0%"
	}
	v := percents[0]
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0%"
	}
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	return formatPercent(v)
}

func detectRAMSpeed() string {
	switch runtime.GOOS {
	case "linux":
		return detectRAMSpeedLinux()
	case "windows":
		return detectRAMSpeedWindows()
	default:
		return ""
	}
}

func detectRAMSpeedLinux() string {
	out, err := exec.Command("dmidecode", "-t", "memory").Output()
	if err != nil {
		return ""
	}
	text := string(out)

	typeRe := regexp.MustCompile(`(?m)^\s*Type:\s*(DDR[0-9A-Za-z\-]+)\s*$`)
	typeMatch := typeRe.FindStringSubmatch(text)

	speedRe := regexp.MustCompile(`(?m)^\s*(?:Configured Memory Speed|Speed):\s*([0-9]+)\s*MT/s\s*$`)
	speedMatches := speedRe.FindAllStringSubmatch(text, -1)
	maxSpeed := 0
	for _, m := range speedMatches {
		if len(m) != 2 {
			continue
		}
		speed, parseErr := strconv.Atoi(m[1])
		if parseErr != nil || speed <= 0 {
			continue
		}
		if speed > maxSpeed {
			maxSpeed = speed
		}
	}

	ramType := ""
	if len(typeMatch) == 2 {
		ramType = strings.TrimSpace(typeMatch[1])
	}

	switch {
	case ramType != "" && maxSpeed > 0:
		return fmt.Sprintf("%s %d MT/s", ramType, maxSpeed)
	case maxSpeed > 0:
		return fmt.Sprintf("%d MT/s", maxSpeed)
	case ramType != "":
		return ramType
	default:
		return ""
	}
}

func detectRAMSpeedWindows() string {
	// Try SMBIOS memory type + speed first.
	ramType := detectRAMTypeWindows()
	speed := detectWindowsMemorySpeed()
	switch {
	case ramType != "" && speed != "":
		return fmt.Sprintf("%s %s", ramType, speed)
	case speed != "":
		return speed
	case ramType != "":
		return ramType
	default:
		return ""
	}
}

func detectWindowsMemorySpeed() string {
	cmds := [][]string{
		{"wmic", "memorychip", "get", "Speed", "/value"},
		{"wmic", "memorychip", "get", "ConfiguredClockSpeed", "/value"},
		{"wmic", "memorychip", "get", "Speed"},
		{"powershell", "-NoProfile", "-Command", "(Get-CimInstance Win32_PhysicalMemory | Select-Object -First 1 -ExpandProperty Speed)"},
	}

	for _, cmd := range cmds {
		if len(cmd) == 0 {
			continue
		}
		out, err := exec.Command(cmd[0], cmd[1:]...).Output()
		if err != nil {
			continue
		}
		if speed := parseWindowsSpeedValue(string(out)); speed != "" {
			return speed
		}
	}
	return ""
}

func parseWindowsSpeedValue(raw string) string {
	re := regexp.MustCompile(`(?m)\b([0-9]{3,5})\b`)
	matches := re.FindAllStringSubmatch(raw, -1)
	maxSpeed := 0
	for _, m := range matches {
		if len(m) != 2 {
			continue
		}
		v, err := strconv.Atoi(m[1])
		if err != nil || v <= 0 {
			continue
		}
		if v > maxSpeed {
			maxSpeed = v
		}
	}
	if maxSpeed == 0 {
		return ""
	}
	return fmt.Sprintf("%d MT/s", maxSpeed)
}

func detectRAMTypeWindows() string {
	cmds := [][]string{
		{"wmic", "memorychip", "get", "SMBIOSMemoryType", "/value"},
		{"wmic", "memorychip", "get", "SMBIOSMemoryType"},
		{"powershell", "-NoProfile", "-Command", "(Get-CimInstance Win32_PhysicalMemory | Select-Object -First 1 -ExpandProperty SMBIOSMemoryType)"},
		{"powershell", "-NoProfile", "-Command", "(Get-CimInstance Win32_PhysicalMemory | Select-Object -First 1 -ExpandProperty MemoryType)"},
	}

	for _, cmd := range cmds {
		if len(cmd) == 0 {
			continue
		}
		out, err := exec.Command(cmd[0], cmd[1:]...).Output()
		if err != nil {
			continue
		}
		if ramType := parseWindowsMemoryType(string(out)); ramType != "" {
			return ramType
		}
	}
	return ""
}

func parseWindowsMemoryType(raw string) string {
	re := regexp.MustCompile(`(?m)\b([0-9]{1,3})\b`)
	matches := re.FindAllStringSubmatch(raw, -1)
	for _, m := range matches {
		if len(m) != 2 {
			continue
		}
		code, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		if ramType := mapSMBIOSMemoryType(code); ramType != "" {
			return ramType
		}
	}
	return ""
}

func mapSMBIOSMemoryType(code int) string {
	switch code {
	case 18:
		return "DDR"
	case 19:
		return "DDR2"
	case 24:
		return "DDR3"
	case 26:
		return "DDR4"
	case 34:
		return "DDR5"
	case 35:
		return "LPDDR"
	case 36:
		return "LPDDR2"
	case 37:
		return "LPDDR3"
	case 38:
		return "LPDDR4"
	case 39:
		return "LPDDR4X"
	case 40:
		return "LPDDR5"
	default:
		return ""
	}
}

func detectStorageDiskName(ctx context.Context, storagePath string) string {
	parts, err := disk.PartitionsWithContext(ctx, false)
	if err != nil || len(parts) == 0 {
		return fallbackStorageDiskName(storagePath)
	}

	cleanPath := filepath.Clean(storagePath)
	bestMount := ""
	bestDevice := ""
	for _, p := range parts {
		mount := filepath.Clean(strings.TrimSpace(p.Mountpoint))
		if mount == "" {
			continue
		}
		if !pathHasPrefix(cleanPath, mount) {
			continue
		}
		if len(mount) > len(bestMount) {
			bestMount = mount
			bestDevice = strings.TrimSpace(p.Device)
		}
	}

	if bestDevice != "" {
		return bestDevice
	}
	return fallbackStorageDiskName(storagePath)
}

func pathHasPrefix(path string, prefix string) bool {
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
		prefix = strings.ToLower(prefix)
	}
	if path == prefix {
		return true
	}
	if strings.HasSuffix(prefix, string(filepath.Separator)) {
		return strings.HasPrefix(path, prefix)
	}
	return strings.HasPrefix(path, prefix+string(filepath.Separator))
}

func fallbackStorageDiskName(storagePath string) string {
	p := strings.TrimSpace(storagePath)
	if p == "" {
		return "Main Disk"
	}
	if runtime.GOOS == "windows" {
		return p
	}
	if p == "/" {
		return "/dev/root"
	}
	return p
}

func formatPercent(v float64) string {
	return fmt.Sprintf("%.0f%%", v)
}

func defaultStoragePath(raw string) string {
	p := strings.TrimSpace(raw)
	if p != "" {
		return p
	}
	if runtime.GOOS == "windows" {
		return "C:\\"
	}
	return "/"
}

func formatBytes(v uint64) string {
	const (
		ki = 1024
		mi = 1024 * ki
		gi = 1024 * mi
		ti = 1024 * gi
	)
	switch {
	case v >= ti:
		return fmt.Sprintf("%.1fTB", float64(v)/float64(ti))
	case v >= gi:
		return fmt.Sprintf("%.1fGB", float64(v)/float64(gi))
	case v >= mi:
		return fmt.Sprintf("%.1fMB", float64(v)/float64(mi))
	case v >= ki:
		return fmt.Sprintf("%.1fKB", float64(v)/float64(ki))
	default:
		return fmt.Sprintf("%dB", v)
	}
}
