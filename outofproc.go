package scopeagent

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/google/uuid"
	"github.com/undefinedlabs/go-agent/tracer"
	"github.com/vmihailenco/msgpack"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

const (
	EnvTestRun      = "SCOPE_TESTRUN"
	EnvForceAgentId = "SCOPE_FORCE_AGENTID"
	EnvForceTraceId = "SCOPE_FORCE_TRACEID"
	EnvForceSpanId  = "SCOPE_FORCE_SPANID"
)

type coverage struct {
	Type    string         `json:"type" msgpack:"type"`
	Version string         `json:"version" msgpack:"version"`
	Uuid    string         `json:"uuid" msgpack:"uuid"`
	Files   []fileCoverage `json:"files" msgpack:"files"`
}
type fileCoverage struct {
	Filename   string  `json:"filename" msgpack:"filename"`
	Boundaries [][]int `json:"boundaries" msgpack:"boundaries"`
}

func checkIfNewTestProcessNeeded(t *testing.T, funcName string) bool {
	if _, exist := os.LookupEnv(EnvTestRun); !exist {
		t.Parallel()

		agentId := GlobalAgent.metadata[AgentID].(string)
		traceId, spanId := tracer.RandomID2()
		traceIdStr := fmt.Sprintf("%x", traceId)
		spanIdStr := fmt.Sprintf("%x", spanId)
		coverageFile := fmt.Sprintf(".%s-%s.cout", agentId, funcName)
		command := os.Args[0]
		var commandArgs []string
		if testing.CoverMode() != "" {
			commandArgs = []string{
				"-test.timeout=30s",
				fmt.Sprintf("-test.run=^(%s)$", funcName),
				fmt.Sprintf("-test.coverprofile=%s", coverageFile),
			}
		} else {
			commandArgs = []string{
				"-test.timeout=30s",
				fmt.Sprintf("-test.run=^(%s)$", funcName),
			}
		}
		cmd := exec.Command(command, commandArgs...)
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, EnvTestRun+"=1")
		cmd.Env = append(cmd.Env, EnvForceAgentId+"="+agentId)
		cmd.Env = append(cmd.Env, EnvForceTraceId+"="+traceIdStr)
		cmd.Env = append(cmd.Env, EnvForceSpanId+"="+spanIdStr)
		output, _ := cmd.CombinedOutput()

		coverageData := getCoverage(coverageFile, true)
		sendCoveragePatch(coverageData, agentId, spanIdStr)

		resString :=  string(output)
		var resArray []string
		tmpArray := strings.Split(resString,"\n")
		for _, line := range tmpArray {
			line = strings.TrimSpace(line)
			if line != "" && line != "PASS" && line != "FAIL" {
				resArray = append(resArray, line)
			}
		}
		resString = strings.Join(resArray, "\n")
		if resString != "" {
			fmt.Println(resString)
		}

		if cmd.ProcessState.ExitCode() == 0 {
			t.SkipNow()
		} else {
			t.FailNow()
		}
		return true
	}
	return false
}

func checkIfFlushNeeded() {
	if _, exist := os.LookupEnv(EnvTestRun); exist {
		GlobalAgent.Stop()
	}
}

func getOutOfProcessContext() (agentId string, traceId uint64, spanId uint64, ok bool) {
	if _, exist := os.LookupEnv(EnvTestRun); exist {
		agentId = os.Getenv(EnvForceAgentId)
		if traceIdStr, set := os.LookupEnv(EnvForceTraceId); set {
			traceId, _ = strconv.ParseUint(traceIdStr, 16, 64)
		}
		if spanIdStr, set := os.LookupEnv(EnvForceSpanId); set {
			spanId, _ = strconv.ParseUint(spanIdStr, 16, 64)
		}
		ok = traceId != 0 && spanId != 0
		return
	} else {
	}
	return "", 0, 0, false
}

func getCoverage(coverageFile string, onlyExecuted bool) *coverage {
	var coverageData *coverage
	fileMap := map[string][][]int{}
	if covFile, covError := os.Open(coverageFile); covError == nil {
		defer os.Remove(coverageFile)
		defer covFile.Close()
		covReader := bufio.NewReader(covFile)
		for {
			line, err := covReader.ReadString('\n')
			if err != nil {
				break
			}
			line = strings.TrimSuffix(line, "\n")
			endFileToken := strings.LastIndex(line, ":")
			file := line[:endFileToken]
			line = line[endFileToken+1:]
			if file == "mode" {
				continue
			}
			endPositionToken := strings.Index(line, " ")
			position := line[:endPositionToken]
			positionsArray := strings.Split(position, ",")
			start := strings.Split(positionsArray[0], ".")
			end := strings.Split(positionsArray[1], ".")

			line = line[endPositionToken+1:]
			remains := strings.Split(line, " ")

			intStartLine, _ := strconv.Atoi(start[0])
			intStartColumn, _ := strconv.Atoi(start[1])
			intEndLine, _ := strconv.Atoi(end[0])
			intEndColumn, _ := strconv.Atoi(end[1])
			intCount, _ := strconv.Atoi(remains[1])
			if onlyExecuted && intCount == 0 {
				continue
			}

			fileMap[file] = append(fileMap[file], []int{
				intStartLine, intStartColumn, intCount,
			})
			fileMap[file] = append(fileMap[file], []int{
				intEndLine, intEndColumn, -1,
			})
		}

		files := make([]fileCoverage, 0)
		for key, value := range fileMap {
			files = append(files, fileCoverage{
				Filename:   key,
				Boundaries: value,
			})
		}

		uuidValue, _ := uuid.NewRandom()
		coverageData = &coverage{
			Type:    "com.undefinedlabs.uccf",
			Version: "0.2.0",
			Uuid:    uuidValue.String(),
			Files:   files,
		}
	}
	return coverageData
}

// Patch transaction
func sendCoveragePatch(coverage *coverage, agentId string, spanId string) {
	if coverage == nil {
		return
	}
	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"agent.id": agentId,
		},
		"spans": []map[string]interface{}{
			{
				"context": map[string]interface{}{
					"span_id": spanId,
				},
				"tags": map[string]interface{}{
					"test.coverage": *coverage,
				},
			},
		},
	}

	binaryPatch, err := msgpack.Marshal(patch)
	if err != nil {
		return
	}

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err = zw.Write(binaryPatch)
	if err != nil {
		return
	}
	if err := zw.Close(); err != nil {
		return
	}
	url := fmt.Sprintf("%s/%s", GlobalAgent.scopeEndpoint, "api/agent/ingest")
	req, err := http.NewRequest("PATCH", url, &buf)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", fmt.Sprintf("scope-agent-go/%s", GlobalAgent.version))
	req.Header.Set("Content-Type", "application/msgpack")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("X-Scope-ApiKey", GlobalAgent.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	//fmt.Printf("%d: %s", resp.StatusCode, resp.Status)
}
