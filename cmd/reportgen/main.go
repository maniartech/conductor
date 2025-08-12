package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	cpu "github.com/shirou/gopsutil/v3/cpu"
	host "github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// Report encapsulates environment, test, race, fuzz, and perf summaries
type Report struct {
	GeneratedAt    time.Time         `json:"generatedAt"`
	GoVersion      string            `json:"goVersion"`
	NumCPU         int               `json:"numCPU"`
	PhysicalCores  int               `json:"physicalCores"`
	GOMAXPROCS     int               `json:"gomaxprocs"`
	HostInfo       *HostInfo         `json:"hostInfo"`
	MemoryInfo     *MemoryInfo       `json:"memoryInfo"`
	CPUInfo        []CPUInfo         `json:"cpuInfo"`
	RaceResult     *CmdResult        `json:"raceResult"`
	UnitResult     *CmdResult        `json:"unitResult"`
	FuzzSeedChecks []CmdResult       `json:"fuzzSeedChecks"`
	PerfMetrics    *PerfMetrics      `json:"perfMetrics"`
	Notes          map[string]string `json:"notes"`
}

type HostInfo struct {
	Hostname  string `json:"hostname"`
	OS        string `json:"os"`
	Platform  string `json:"platform"`
	PlatformV string `json:"platformVersion"`
	Kernel    string `json:"kernelVersion"`
	Uptime    uint64 `json:"uptimeSeconds"`
}

type MemoryInfo struct {
	Total       uint64  `json:"totalBytes"`
	Available   uint64  `json:"availableBytes"`
	Used        uint64  `json:"usedBytes"`
	UsedPercent float64 `json:"usedPercent"`
}

type CPUInfo struct {
	ModelName string  `json:"modelName"`
	Cores     int32   `json:"cores"`
	Mhz       float64 `json:"mhz"`
}

type CmdResult struct {
	Command string `json:"command"`
	OK      bool   `json:"ok"`
	Output  string `json:"output"`
	TimeMS  int64  `json:"timeMs"`
}

type PerfMetrics struct {
	ThroughputTPS       float64 `json:"throughputTasksPerSec"`
	MemoryGrowthBytes   int64   `json:"memoryGrowthBytes"`
	GoroutineLeakGrowth int     `json:"goroutineLeakGrowth"`
}

func runCmd(ctx context.Context, args ...string) *CmdResult {
	cmdStr := strings.Join(args, " ")
	start := time.Now()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	elapsed := time.Since(start)
	return &CmdResult{
		Command: cmdStr,
		OK:      err == nil,
		Output:  buf.String(),
		TimeMS:  elapsed.Milliseconds(),
	}
}

func collectHost() (*HostInfo, *MemoryInfo, []CPUInfo) {
	h, _ := host.Info()
	m, _ := mem.VirtualMemory()
	ci, _ := cpu.Info()
	hostInfo := &HostInfo{}
	if h != nil {
		hostInfo.Hostname = valueOr("", h.Hostname)
		hostInfo.OS = runtime.GOOS
		hostInfo.Platform = valueOr("", h.Platform)
		hostInfo.PlatformV = valueOr("", h.PlatformVersion)
		hostInfo.Kernel = valueOr("", h.KernelVersion)
		hostInfo.Uptime = h.Uptime
	}
	memInfo := &MemoryInfo{}
	if m != nil {
		memInfo.Total = m.Total
		memInfo.Available = m.Available
		memInfo.Used = m.Used
		memInfo.UsedPercent = m.UsedPercent
	}
	var cpuList []CPUInfo
	for _, c := range ci {
		cpuList = append(cpuList, CPUInfo{ModelName: c.ModelName, Cores: c.Cores, Mhz: c.Mhz})
	}
	return hostInfo, memInfo, cpuList
}

func valueOr(def string, v string) string {
	if v == "" {
		return def
	}
	return v
}

func main() {
	var out string
	var fuzzSeed string
	var skipPerf bool
	var format string
	var updateReport string
	var startMarker string
	var endMarker string
	flag.StringVar(&out, "out", "report.json", "output file path (json or md depending on -format)")
	flag.StringVar(&fuzzSeed, "fuzz-seed", "FuzzComplexDataTypes/5a4c303d10b86c14", "fuzz seed to re-run (package-qualified)")
	flag.BoolVar(&skipPerf, "skip-perf", false, "skip performance test runs")
	flag.StringVar(&format, "format", "json", "output format: json|markdown|both")
	flag.StringVar(&updateReport, "update-report", "", "path to CURRENT_STATUS_REPORT.md to embed markdown output")
	flag.StringVar(&startMarker, "start-marker", "<!-- AUTO:START:REPORT -->", "start marker to replace in report file")
	flag.StringVar(&endMarker, "end-marker", "<!-- AUTO:END:REPORT -->", "end marker to replace in report file")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	hostInfo, memInfo, cpuList := collectHost()
	physCores, _ := cpu.Counts(false)

	rep := &Report{
		GeneratedAt:   time.Now(),
		GoVersion:     runtime.Version(),
		NumCPU:        runtime.NumCPU(),
		PhysicalCores: physCores,
		GOMAXPROCS:    runtime.GOMAXPROCS(0),
		HostInfo:      hostInfo,
		MemoryInfo:    memInfo,
		CPUInfo:       cpuList,
		Notes:         map[string]string{},
	}

	// Run unit/integration
	rep.UnitResult = runCmd(ctx, "go", "test", "-v", "./...")

	// Run race suite
	rep.RaceResult = runCmd(ctx, "go", "test", "-race", "-count=1", "./...")

	// Run specific fuzz seed (without fuzzing engine)
	if fuzzSeed != "" {
		rep.FuzzSeedChecks = append(rep.FuzzSeedChecks, *runCmd(ctx, "go", "test", "-run="+fuzzSeed, "-v", "./..."))
	}

	// Optional: lightweight perf extraction from logs (heuristic)
	if !skipPerf {
		// High-throughput stress test
		ht := runCmd(ctx, "go", "test", "-run", "TestProductionStress_HighThroughput", "-v", "./...")
		// Memory stability
		ms := runCmd(ctx, "go", "test", "-run", "TestProductionStress_MemoryStability", "-v", "./...")
		// Goroutine lifecycle
		gl := runCmd(ctx, "go", "test", "-run", "TestProductionStress_GoroutineLifecycle", "-v", "./...")

		perf := &PerfMetrics{}
		perf.ThroughputTPS = parseThroughput(ht.Output)
		perf.MemoryGrowthBytes = parseMemoryGrowth(ms.Output)
		perf.GoroutineLeakGrowth = parseGoroutineGrowth(gl.Output)
		rep.PerfMetrics = perf
	}

	// Write outputs
	switch strings.ToLower(format) {
	case "json":
		writeJSON(out, rep)
	case "markdown":
		writeMarkdown(out, rep)
	case "both":
		writeJSON(out, rep)
		md := strings.TrimSuffix(out, ".json") + ".md"
		writeMarkdown(md, rep)
	default:
		writeJSON(out, rep)
	}

	// Optionally embed markdown into a status report file between markers
	if updateReport != "" {
		mdContent := buildMarkdown(rep)
		if err := replaceInFile(updateReport, startMarker, endMarker, mdContent); err != nil {
			fmt.Fprintf(os.Stderr, "failed to update report file: %v\n", err)
		} else {
			fmt.Printf("Updated report section in %s between markers.\n", updateReport)
		}
	}

	fmt.Printf("Report generated: %s (format=%s)\n", out, format)
}

func writeJSON(path string, rep *Report) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rep); err != nil {
		panic(err)
	}
}

func writeMarkdown(path string, rep *Report) {
	content := buildMarkdown(rep)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		panic(err)
	}
}

func buildMarkdown(rep *Report) string {
	var b strings.Builder
	b.WriteString("# Automated Test & Performance Report\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\\n\\n", rep.GeneratedAt.Format(time.RFC3339)))

	b.WriteString("## Environment\n\n")
	b.WriteString(fmt.Sprintf("- Go: %s\\n", rep.GoVersion))
	b.WriteString(fmt.Sprintf("- Num CPU (logical): %d\\n", rep.NumCPU))
	b.WriteString(fmt.Sprintf("- Physical Cores: %d\\n", rep.PhysicalCores))
	b.WriteString(fmt.Sprintf("- GOMAXPROCS: %d\\n", rep.GOMAXPROCS))
	if rep.HostInfo != nil {
		b.WriteString(fmt.Sprintf("- Hostname: %s\\n", rep.HostInfo.Hostname))
		b.WriteString(fmt.Sprintf("- OS/Platform: %s / %s %s\\n", rep.HostInfo.OS, rep.HostInfo.Platform, rep.HostInfo.PlatformV))
		b.WriteString(fmt.Sprintf("- Kernel: %s\\n", rep.HostInfo.Kernel))
		b.WriteString(fmt.Sprintf("- Uptime: %ds\\n", rep.HostInfo.Uptime))
	}
	if rep.MemoryInfo != nil {
		b.WriteString(fmt.Sprintf("- Memory: total=%d, used=%d (%.2f%%), available=%d\\n", rep.MemoryInfo.Total, rep.MemoryInfo.Used, rep.MemoryInfo.UsedPercent, rep.MemoryInfo.Available))
	}
	if len(rep.CPUInfo) > 0 {
		b.WriteString("- CPU(s):\\n")
		for i, c := range rep.CPUInfo {
			b.WriteString(fmt.Sprintf("  - [%d] %s, cores=%d, mhz=%.0f\\n", i, c.ModelName, c.Cores, c.Mhz))
		}
	}
	b.WriteString("\n---\n\n")

	b.WriteString("## Test Results\n\n")
	if rep.UnitResult != nil {
		b.WriteString(fmt.Sprintf("- Unit/Integration: %s (%dms)\\n", okStr(rep.UnitResult.OK), rep.UnitResult.TimeMS))
	}
	if rep.RaceResult != nil {
		b.WriteString(fmt.Sprintf("- Race Detector: %s (%dms)\\n", okStr(rep.RaceResult.OK), rep.RaceResult.TimeMS))
	}
	if len(rep.FuzzSeedChecks) > 0 {
		for _, f := range rep.FuzzSeedChecks {
			b.WriteString(fmt.Sprintf("- Fuzz Seed: %s => %s (%dms)\\n", f.Command, okStr(f.OK), f.TimeMS))
		}
	}
	b.WriteString("\n---\n\n")

	b.WriteString("## Performance Metrics\n\n")
	if rep.PerfMetrics != nil {
		b.WriteString(fmt.Sprintf("- Throughput: %.0f tasks/sec\\n", rep.PerfMetrics.ThroughputTPS))
		b.WriteString(fmt.Sprintf("- Memory Growth: %d bytes\\n", rep.PerfMetrics.MemoryGrowthBytes))
		b.WriteString(fmt.Sprintf("- Goroutine Growth: %d\\n", rep.PerfMetrics.GoroutineLeakGrowth))
	} else {
		b.WriteString("(perf skipped)\\n")
	}
	return b.String()
}

func replaceInFile(path, startMarker, endMarker, content string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(data)
	startIdx := strings.Index(text, startMarker)
	endIdx := strings.Index(text, endMarker)
	if startIdx >= 0 && endIdx > startIdx {
		newText := text[:startIdx+len(startMarker)] + "\n\n" + content + "\n\n" + text[endIdx:]
		return os.WriteFile(path, []byte(newText), 0o644)
	}
	// markers not found: append at end with markers
	appendText := "\n\n" + startMarker + "\n\n" + content + "\n\n" + endMarker + "\n"
	return os.WriteFile(path, append([]byte(text), []byte(appendText)...), 0o644)
}

func okStr(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}

var (
	throughputRe = regexp.MustCompile(`Throughput:\s*([0-9,\.]+)\s*tasks/second`)
	memGrowthRe  = regexp.MustCompile(`Memory growth:\s*([0-9,]+)\s*bytes`)
	growthRe     = regexp.MustCompile(`(?i)(Final growth|Growth):\s*([0-9]+)`) // pick last occurrence
)

func parseThroughput(s string) float64 {
	matches := throughputRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return 0
	}
	last := matches[len(matches)-1]
	val := strings.ReplaceAll(last[1], ",", "")
	v, _ := strconv.ParseFloat(val, 64)
	return v
}

func parseMemoryGrowth(s string) int64 {
	matches := memGrowthRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return 0
	}
	last := matches[len(matches)-1]
	val := strings.ReplaceAll(last[1], ",", "")
	v, _ := strconv.ParseInt(val, 10, 64)
	return v
}

func parseGoroutineGrowth(s string) int {
	matches := growthRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return 0
	}
	last := matches[len(matches)-1]
	v, _ := strconv.Atoi(last[2])
	return v
}
