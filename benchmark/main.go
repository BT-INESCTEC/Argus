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
	DSTAT_PRE_DELAY   = 10 * time.Second
	DSTAT_POST_DELAY  = 10 * time.Second
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
}

var workflowFiles = []WorkflowFile{
	{Path: "../Argus_artifacts/VWBench/.github/workflows/1.yml", Name: "vwbench_workflow1"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/2.yml", Name: "vwbench_workflow2"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/3.yml", Name: "vwbench_workflow3"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/4.yml", Name: "vwbench_workflow4"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/5.yml", Name: "vwbench_workflow5"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/6.yml", Name: "vwbench_workflow6"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/7.yml", Name: "vwbench_workflow7"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/8.yml", Name: "vwbench_workflow8"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/9.yml", Name: "vwbench_workflow9"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/10.yml", Name: "vwbench_workflow10"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/11.yml", Name: "vwbench_workflow11"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/12.yml", Name: "vwbench_workflow12"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/13.yml", Name: "vwbench_workflow13"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/14.yml", Name: "vwbench_workflow14"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/15.yml", Name: "vwbench_workflow15"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/16.yml", Name: "vwbench_workflow16"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/17.yml", Name: "vwbench_workflow17"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/18.yml", Name: "vwbench_workflow18"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/19.yml", Name: "vwbench_workflow19"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/20.yml", Name: "vwbench_workflow20"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/21.yml", Name: "vwbench_workflow21"},
	// {Path: "../Argus_artifacts/VWBench/.github/workflows/22.yml", Name: "vwbench_workflow22"},
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

	resultsFile := filepath.Join(resultsDir, "benchmark_results.csv")
	csvFile, err := os.Create(resultsFile)
	if err != nil {
		log.Fatalf("Failed to create results file: %v", err)
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

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
	if err := writer.Write(header); err != nil {
		log.Fatalf("Failed to write CSV header: %v", err)
	}

	totalRuns := len(workflowFiles) * RUNS_PER_WORKFLOW
	currentRun := 0

	for _, workflow := range workflowFiles {
		log.Printf("\n📁 Testing workflow: %s", workflow.Name)

		for run := 1; run <= RUNS_PER_WORKFLOW; run++ {
			currentRun++
			log.Printf("  [%d/%d] Run %d/%d", currentRun, totalRuns, run, RUNS_PER_WORKFLOW)

			result, err := runBenchmark(workflow, run, rawDstatDir)
			if err != nil {
				log.Printf("    ❌ Error: %v", err)
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
			if err := writer.Write(record); err != nil {
				log.Printf("    ⚠️  Failed to write result: %v", err)
			}
			writer.Flush()

			log.Printf("    ✅ Completed in %.2fs", result.ExecutionTime)

			time.Sleep(2 * time.Second)
		}
	}

	log.Printf("\n✨ Benchmarking complete! Results saved to: %s\n", resultsFile)
}

func runBenchmark(workflow WorkflowFile, runNumber int, rawDstatDir string) (*BenchmarkResult, error) {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	outputFile := filepath.Join("results", fmt.Sprintf("%s_run%d.sarif", workflow.Name, runNumber))

	var dstatPID int
	var dstatCmd *exec.Cmd

	if ENABLE_DSTAT {
		dstatFile := filepath.Join(rawDstatDir, fmt.Sprintf("%s_run%d_%s.csv", workflow.Name, runNumber, timestamp))

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
	argusCmd := exec.Command("poetry", "run", "python3", "argus.py",
		"--mode", "file",
		"--file", workflow.Path,
		"--output", outputFile)
	argusCmd.Dir = ".."
	argusCmd.Stdout = nil
	argusCmd.Stderr = nil

	println(argusCmd.Dir)

	if err := argusCmd.Run(); err != nil {
		return nil, fmt.Errorf("argus failed: %w", err)
	}
	executionTime := time.Since(startTime).Seconds()

	result := &BenchmarkResult{
		WorkflowFile:  workflow.Name,
		RunNumber:     runNumber,
		ExecutionTime: executionTime,
		Timestamp:     timestamp,
	}

	if ENABLE_DSTAT {
		time.Sleep(DSTAT_POST_DELAY)

		if p, err := os.FindProcess(dstatPID); err == nil {
			p.Kill()
		}
		dstatCmd.Wait()

		dstatFile := filepath.Join(rawDstatDir, fmt.Sprintf("%s_run%d_%s.csv", workflow.Name, runNumber, timestamp))
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
