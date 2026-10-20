package collector

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var preferredHwmonNames = map[string]int{
	"coretemp":    0,
	"k10temp":     1,
	"cpu_thermal": 2,
	"soc_thermal": 3,
	"acpitz":      4,
}

type LinuxHwmonProvider struct {
	root string
}

func NewLinuxHwmonProvider(root string) *LinuxHwmonProvider {
	if strings.TrimSpace(root) == "" {
		root = "/sys/class/hwmon"
	}
	return &LinuxHwmonProvider{root: root}
}

func (p *LinuxHwmonProvider) TemperatureC(_ context.Context) (float64, error) {
	dirs, err := filepath.Glob(filepath.Join(p.root, "hwmon*"))
	if err != nil {
		return 0, fmt.Errorf("scan hwmon dirs: %w", err)
	}
	if len(dirs) == 0 {
		return 0, errors.New("no hwmon dirs")
	}

	entries := make([]hwmonReading, 0, 16)
	for _, dir := range dirs {
		name := strings.TrimSpace(readTextFile(filepath.Join(dir, "name")))
		tempFiles, _ := filepath.Glob(filepath.Join(dir, "temp*_input"))
		for _, tempFile := range tempFiles {
			raw := strings.TrimSpace(readTextFile(tempFile))
			if raw == "" {
				continue
			}
			milliC, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				continue
			}
			celsius := milliC / 1000.0
			if celsius < -40 || celsius > 200 {
				continue
			}
			entries = append(entries, hwmonReading{name: name, tempC: celsius})
		}
	}
	if len(entries) == 0 {
		return 0, errors.New("no valid hwmon temperature")
	}

	sort.Slice(entries, func(i, j int) bool {
		ri := scoreHwmon(entries[i].name)
		rj := scoreHwmon(entries[j].name)
		if ri != rj {
			return ri < rj
		}
		return entries[i].tempC > entries[j].tempC
	})

	return entries[0].tempC, nil
}

func (p *LinuxHwmonProvider) FanRPM(_ context.Context) (int, error) {
	dirs, err := filepath.Glob(filepath.Join(p.root, "hwmon*"))
	if err != nil {
		return 0, fmt.Errorf("scan hwmon dirs: %w", err)
	}
	if len(dirs) == 0 {
		return 0, errors.New("no hwmon dirs")
	}

	type fanReading struct {
		name string
		rpm  int
	}
	readings := make([]fanReading, 0, 8)
	for _, dir := range dirs {
		name := strings.TrimSpace(readTextFile(filepath.Join(dir, "name")))
		fanFiles, _ := filepath.Glob(filepath.Join(dir, "fan*_input"))
		for _, fanFile := range fanFiles {
			raw := strings.TrimSpace(readTextFile(fanFile))
			if raw == "" {
				continue
			}
			rpm, parseErr := strconv.Atoi(raw)
			if parseErr != nil || rpm <= 0 || rpm > 20000 {
				continue
			}
			readings = append(readings, fanReading{name: name, rpm: rpm})
		}
	}
	if len(readings) == 0 {
		return 0, errors.New("no valid fan reading")
	}

	sort.Slice(readings, func(i, j int) bool {
		ri := scoreHwmon(readings[i].name)
		rj := scoreHwmon(readings[j].name)
		if ri != rj {
			return ri < rj
		}
		return readings[i].rpm > readings[j].rpm
	})
	return readings[0].rpm, nil
}

type hwmonReading struct {
	name  string
	tempC float64
}

func scoreHwmon(name string) int {
	if rank, ok := preferredHwmonNames[strings.ToLower(strings.TrimSpace(name))]; ok {
		return rank
	}
	return 100
}

func readTextFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}
