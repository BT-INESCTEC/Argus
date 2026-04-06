package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	RUNS_PER_WORKFLOW = 3
	ENABLE_DSTAT      = true
	DSTAT_PRE_DELAY   = 1 * time.Second
	DSTAT_POST_DELAY  = 1 * time.Second
	BASELINE_DURATION = 10 * time.Second
)

type WorkflowFile struct {
	Path string
	Name string
}

type BenchmarkResult struct {
	WorkflowFile  string
	RunNumber     int
	ExecutionTime float64
	AvgCPU        float64
	PeakMemory    float64
	AvgDiskRead   float64
	AvgDiskWrite  float64
	AvgNetRecv    float64
	AvgNetSend    float64
	Timestamp     string
	Tool          string
}

type BaselineResult struct {
	AvgCPU       float64
	PeakMemory   float64
	AvgDiskRead  float64
	AvgDiskWrite float64
	AvgNetRecv   float64
	AvgNetSend   float64
	Timestamp    string
}

var workflowFiles = []WorkflowFile{
	{Path: "../Argus_artifacts/VWBench/.github/workflows/1.yml", Name: "vwbench_workflow1"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/2.yml", Name: "vwbench_workflow2"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/3.yml", Name: "vwbench_workflow3"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/4.yml", Name: "vwbench_workflow4"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/5.yml", Name: "vwbench_workflow5"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/6.yml", Name: "vwbench_workflow6"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/7.yml", Name: "vwbench_workflow7"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/8.yml", Name: "vwbench_workflow8"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/9.yml", Name: "vwbench_workflow9"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/10.yml", Name: "vwbench_workflow10"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/11.yml", Name: "vwbench_workflow11"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/12.yml", Name: "vwbench_workflow12"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/13.yml", Name: "vwbench_workflow13"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/14.yml", Name: "vwbench_workflow14"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/15.yml", Name: "vwbench_workflow15"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/16.yml", Name: "vwbench_workflow16"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/17.yml", Name: "vwbench_workflow17"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/18.yml", Name: "vwbench_workflow18"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/19.yml", Name: "vwbench_workflow19"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/20.yml", Name: "vwbench_workflow20"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/21.yml", Name: "vwbench_workflow21"},
	{Path: "../Argus_artifacts/VWBench/.github/workflows/22.yml", Name: "vwbench_workflow22"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/23.yml", Name: "vwbench_workflow23"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/24.yml", Name: "vwbench_workflow24"},
}

func main() {
	log.Println("Starting Argus Benchmarking Suite")
	log.Printf("Testing %d workflows with %d runs each\n", len(workflowFiles), RUNS_PER_WORKFLOW)
	if ENABLE_DSTAT {
		log.Println("dstat resource monitoring: ENABLED")
	} else {
		log.Println("dstat resource monitoring: DISABLED (timing only)")
	}

	resultsDir := "results"
	rawDstatDir := filepath.Join(resultsDir, "raw_dstat")
	os.MkdirAll(rawDstatDir, 0755)

	// Create output directory for SARIF results with subdirectories for each workflow
	sarifOutputDir := filepath.Join(resultsDir, "sarif_outputs")
	os.MkdirAll(sarifOutputDir, 0755)
	for _, workflow := range workflowFiles {
		workflowOutputDir := filepath.Join(sarifOutputDir, workflow.Name)
		os.MkdirAll(workflowOutputDir, 0755)
	}

	if ENABLE_DSTAT {
		log.Printf("\n📊 Collecting baseline statistics (%v idle)...", BASELINE_DURATION)
		baseline, err := collectBaseline(rawDstatDir)
		if err != nil {
			log.Printf("  ⚠️  Failed to collect baseline: %v", err)
		} else {
			baselineFile := filepath.Join(resultsDir, "baseline_results.csv")
			if err := writeBaselineCSV(baselineFile, baseline); err != nil {
				log.Printf("  ⚠️  Failed to write baseline CSV: %v", err)
			} else {
				log.Printf("  ✅ Baseline collected. Results saved to: %s", baselineFile)
			}
		}
	}

	resultsFileArgus := filepath.Join(resultsDir, "benchmark_results_argus.csv")
	csvFileArgus, err := os.Create(resultsFileArgus)
	if err != nil {
		log.Fatalf("Failed to create Argus results file: %v", err)
	}
	defer csvFileArgus.Close()
	writerArgus := csv.NewWriter(csvFileArgus)
	defer writerArgus.Flush()

	resultsFileZizmor := filepath.Join(resultsDir, "benchmark_results_zizmor.csv")
	csvFileZizmor, err := os.Create(resultsFileZizmor)
	if err != nil {
		log.Fatalf("Failed to create zizmor results file: %v", err)
	}
	defer csvFileZizmor.Close()
	writerZizmor := csv.NewWriter(csvFileZizmor)
	defer writerZizmor.Flush()

	var header []string
	if ENABLE_DSTAT {
		header = []string{
			"workflow_file", "run_number", "execution_time_seconds",
			"avg_cpu_percent", "peak_memory_mb", "avg_disk_read_kb", "avg_disk_write_kb",
			"avg_net_recv_kb", "avg_net_send_kb", "timestamp",
		}
	} else {
		header = []string{
			"workflow_file", "run_number", "execution_time_seconds", "timestamp",
		}
	}
	if err := writerArgus.Write(header); err != nil {
		log.Fatalf("Failed to write Argus CSV header: %v", err)
	}
	if err := writerZizmor.Write(header); err != nil {
		log.Fatalf("Failed to write zizmor CSV header: %v", err)
	}

	totalRuns := len(workflowFiles) * RUNS_PER_WORKFLOW
	currentRun := 0

	for _, workflow := range workflowFiles {
		log.Printf("\n📁 Testing workflow: %s", workflow.Name)

		// Get output directory for this workflow
		workflowOutputDir := filepath.Join(sarifOutputDir, workflow.Name)

		for run := 1; run <= RUNS_PER_WORKFLOW; run++ {
			currentRun++
			log.Printf("  [%d/%d] Run %d/%d - Argus", currentRun, totalRuns*2, run, RUNS_PER_WORKFLOW)

			result, err := runBenchmark(workflow, run, rawDstatDir, "argus", workflowOutputDir)
			if err != nil {
				log.Printf("    ❌ Argus Error: %v", err)
				continue
			}

			var record []string
			if ENABLE_DSTAT {
				record = []string{
					result.WorkflowFile,
					strconv.Itoa(result.RunNumber),
					fmt.Sprintf("%.3f", result.ExecutionTime),
					fmt.Sprintf("%.2f", result.AvgCPU),
					fmt.Sprintf("%.2f", result.PeakMemory),
					fmt.Sprintf("%.2f", result.AvgDiskRead),
					fmt.Sprintf("%.2f", result.AvgDiskWrite),
					fmt.Sprintf("%.2f", result.AvgNetRecv),
					fmt.Sprintf("%.2f", result.AvgNetSend),
					result.Timestamp,
				}
			} else {
				record = []string{
					result.WorkflowFile,
					strconv.Itoa(result.RunNumber),
					fmt.Sprintf("%.3f", result.ExecutionTime),
					result.Timestamp,
				}
			}

			if result.Tool == "argus" {
				if err := writerArgus.Write(record); err != nil {
					log.Printf("    ⚠️  Failed to write Argus result: %v", err)
				}
				writerArgus.Flush()
			} else if result.Tool == "zizmor" {
				if err := writerZizmor.Write(record); err != nil {
					log.Printf("    ⚠️  Failed to write zizmor result: %v", err)
				}
				writerZizmor.Flush()
			}

			log.Printf("    ✅ Argus completed in %.2fs", result.ExecutionTime)

			// Run zizmor on the same workflow
			log.Printf("  [%d/%d] Run %d/%d - zizmor", currentRun+1, totalRuns*2, run, RUNS_PER_WORKFLOW)
			resultZizmor, err := runBenchmark(workflow, run, rawDstatDir, "zizmor", workflowOutputDir)
			if err != nil {
				log.Printf("    ❌ zizmor Error: %v", err)
			} else {
				var recordZizmor []string
				if ENABLE_DSTAT {
					recordZizmor = []string{
						resultZizmor.WorkflowFile,
						strconv.Itoa(resultZizmor.RunNumber),
						fmt.Sprintf("%.3f", resultZizmor.ExecutionTime),
						fmt.Sprintf("%.2f", resultZizmor.AvgCPU),
						fmt.Sprintf("%.2f", resultZizmor.PeakMemory),
						fmt.Sprintf("%.2f", resultZizmor.AvgDiskRead),
						fmt.Sprintf("%.2f", resultZizmor.AvgDiskWrite),
						fmt.Sprintf("%.2f", resultZizmor.AvgNetRecv),
						fmt.Sprintf("%.2f", resultZizmor.AvgNetSend),
						resultZizmor.Timestamp,
					}
				} else {
					recordZizmor = []string{
						resultZizmor.WorkflowFile,
						strconv.Itoa(resultZizmor.RunNumber),
						fmt.Sprintf("%.3f", resultZizmor.ExecutionTime),
						resultZizmor.Timestamp,
					}
				}
				if err := writerZizmor.Write(recordZizmor); err != nil {
					log.Printf("    ⚠️  Failed to write zizmor result: %v", err)
				}
				writerZizmor.Flush()
				log.Printf("    ✅ zizmor completed in %.2fs", resultZizmor.ExecutionTime)
			}

			time.Sleep(2 * time.Second)
		}
	}

	log.Printf("\n✨ Benchmarking complete!")
	log.Printf("   Argus results saved to: %s", resultsFileArgus)
	log.Printf("   zizmor results saved to: %s\n", resultsFileZizmor)
	log.Printf("   SARIF outputs saved to: %s\n", sarifOutputDir)
}

func runBenchmark(workflow WorkflowFile, runNumber int, rawDstatDir string, tool string, outputDir string) (*BenchmarkResult, error) {
	timestamp := time.Now().Format("2006-01-02_15-04-05")

	var dstatPID int
	var dstatCmd *exec.Cmd

	if ENABLE_DSTAT {
		dstatFile := filepath.Join(rawDstatDir, fmt.Sprintf("%s_%s_run%d_%s.csv", workflow.Name, tool, runNumber, timestamp))

		dstatCmd = exec.Command("dstat",
			"--time", "--cpu", "--mem", "--net", "--disk", "--swap",
			"--output", dstatFile)
		dstatCmd.Stdout = nil
		dstatCmd.Stderr = nil

		if err := dstatCmd.Start(); err != nil {
			return nil, fmt.Errorf("failed to start dstat: %w", err)
		}
		dstatPID = dstatCmd.Process.Pid
		defer func() {
			if p, err := os.FindProcess(dstatPID); err == nil {
				p.Kill()
			}
		}()

		time.Sleep(DSTAT_PRE_DELAY)
	}

	startTime := time.Now()

	var cmd *exec.Cmd
	if tool == "argus" {
		outputFile, err := filepath.Abs(filepath.Join(outputDir, "argus.sarif"))
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for output file: %w", err)
		}
		cmd = exec.Command("poetry", "run", "python3", "argus.py",
			"--mode", "file",
			"--file", workflow.Path,
			"--output", outputFile)
		cmd.Dir = ".."
	} else if tool == "zizmor" {
		// zizmor runs from current directory (benchmark), so we need to adjust the path
		// The workflow.Path is relative to Argus root (../Argus_artifacts/...)
		// From benchmark dir, we need ../../Argus_artifacts/...
		zizmorPath := filepath.Join("..", workflow.Path)
		outputFile := filepath.Join(outputDir, "zizmor.sarif")
		cmd = exec.Command("zizmor",
			"--format", "sarif",
			zizmorPath)
		// Redirect stdout to the output file
		stdoutFile, err := os.Create(outputFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create output file: %w", err)
		}
		defer stdoutFile.Close()
		cmd.Stdout = stdoutFile
	} else {
		return nil, fmt.Errorf("unknown tool: %s", tool)
	}
	// Capture stderr for debugging
	var stderrBuf strings.Builder
	if tool != "zizmor" {
		cmd.Stdout = nil
	}
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		stderrOutput := stderrBuf.String()
		if stderrOutput != "" {
			return nil, fmt.Errorf("%s failed: %w (stderr: %s)", tool, err, stderrOutput[:min(len(stderrOutput), 200)])
		}
		return nil, fmt.Errorf("%s failed: %w", tool, err)
	}
	executionTime := time.Since(startTime).Seconds()

	result := &BenchmarkResult{
		WorkflowFile:  workflow.Name,
		RunNumber:     runNumber,
		ExecutionTime: executionTime,
		Timestamp:     timestamp,
		Tool:          tool,
	}

	if ENABLE_DSTAT {
		time.Sleep(DSTAT_POST_DELAY)

		if p, err := os.FindProcess(dstatPID); err == nil {
			p.Kill()
		}
		dstatCmd.Wait()

		time.Sleep(500 * time.Millisecond)

		dstatFile := filepath.Join(rawDstatDir, fmt.Sprintf("%s_%s_run%d_%s.csv", workflow.Name, tool, runNumber, timestamp))
		metrics, err := parseDstatOutput(dstatFile)
		if err != nil {
			return nil, fmt.Errorf("failed to parse dstat: %w", err)
		}

		result.AvgCPU = metrics["avg_cpu"]
		result.PeakMemory = metrics["peak_memory"]
		result.AvgDiskRead = metrics["avg_disk_read"]
		result.AvgDiskWrite = metrics["avg_disk_write"]
		result.AvgNetRecv = metrics["avg_net_recv"]
		result.AvgNetSend = metrics["avg_net_send"]
	}

	return result, nil
}

func parseDstatOutput(filePath string) (map[string]float64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	dataStart := 6
	if len(records) <= dataStart {
		return nil, fmt.Errorf("no data found in dstat output")
	}

	var cpuValues, memValues, diskReadValues, diskWriteValues, netRecvValues, netSendValues []float64

	for i := dataStart; i < len(records); i++ {
		record := records[i]
		if len(record) < 14 {
			continue
		}

		if cpuUsr, err := parseFloat(record[1]); err == nil {
			if cpuSys, err := parseFloat(record[2]); err == nil {
				cpuValues = append(cpuValues, cpuUsr+cpuSys)
			}
		}

		if mem, err := parseFloat(record[6]); err == nil {
			memValues = append(memValues, mem/1024/1024)
		}

		if diskRead, err := parseFloat(record[12]); err == nil {
			diskReadValues = append(diskReadValues, diskRead/1024)
		}
		if diskWrite, err := parseFloat(record[13]); err == nil {
			diskWriteValues = append(diskWriteValues, diskWrite/1024)
		}

		if netRecv, err := parseFloat(record[10]); err == nil {
			netRecvValues = append(netRecvValues, netRecv/1024)
		}
		if netSend, err := parseFloat(record[11]); err == nil {
			netSendValues = append(netSendValues, netSend/1024)
		}
	}

	metrics := map[string]float64{
		"avg_cpu":        average(cpuValues),
		"peak_memory":    max(memValues),
		"avg_disk_read":  average(diskReadValues),
		"avg_disk_write": average(diskWriteValues),
		"avg_net_recv":   average(netRecvValues),
		"avg_net_send":   average(netSendValues),
	}

	return metrics, nil
}

func collectBaseline(rawDstatDir string) (*BaselineResult, error) {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	dstatFile := filepath.Join(rawDstatDir, fmt.Sprintf("baseline_%s.csv", timestamp))

	dstatCmd := exec.Command("dstat",
		"--time", "--cpu", "--mem", "--net", "--disk", "--swap",
		"--output", dstatFile)
	dstatCmd.Stdout = nil
	dstatCmd.Stderr = nil

	if err := dstatCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start dstat for baseline: %w", err)
	}
	dstatPID := dstatCmd.Process.Pid

	time.Sleep(BASELINE_DURATION)

	if p, err := os.FindProcess(dstatPID); err == nil {
		p.Kill()
	}
	dstatCmd.Wait()

	time.Sleep(500 * time.Millisecond)

	metrics, err := parseDstatOutput(dstatFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse baseline dstat: %w", err)
	}

	return &BaselineResult{
		AvgCPU:       metrics["avg_cpu"],
		PeakMemory:   metrics["peak_memory"],
		AvgDiskRead:  metrics["avg_disk_read"],
		AvgDiskWrite: metrics["avg_disk_write"],
		AvgNetRecv:   metrics["avg_net_recv"],
		AvgNetSend:   metrics["avg_net_send"],
		Timestamp:    timestamp,
	}, nil
}

func writeBaselineCSV(filePath string, baseline *BaselineResult) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"avg_cpu_percent", "peak_memory_mb",
		"avg_disk_read_kb", "avg_disk_write_kb",
		"avg_net_recv_kb", "avg_net_send_kb",
		"timestamp",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	record := []string{
		fmt.Sprintf("%.2f", baseline.AvgCPU),
		fmt.Sprintf("%.2f", baseline.PeakMemory),
		fmt.Sprintf("%.2f", baseline.AvgDiskRead),
		fmt.Sprintf("%.2f", baseline.AvgDiskWrite),
		fmt.Sprintf("%.2f", baseline.AvgNetRecv),
		fmt.Sprintf("%.2f", baseline.AvgNetSend),
		baseline.Timestamp,
	}
	return w.Write(record)
}

func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	return strconv.ParseFloat(s, 64)
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	maxVal := values[0]
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
