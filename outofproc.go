package scopeagent

import (
	"bufio"
	"fmt"
	"github.com/google/uuid"
	"github.com/undefinedlabs/go-agent/tracer"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

const(
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

		coverageFile := fmt.Sprintf(".%s-%s.cout", agentId, funcName)
		command := os.Args[0]
		commandArgs := []string{
			"-test.timeout=30s",
			fmt.Sprintf("-test.run=^(%s)$", funcName),
			fmt.Sprintf("-test.coverprofile=%s", coverageFile),
		}
		fmt.Printf("Executing test: %s\n", funcName)
		cmd := exec.Command(command, commandArgs...)
		cmd.Env = append(cmd.Env, EnvTestRun+ "=1")
		cmd.Env = append(cmd.Env, EnvForceAgentId+ "=" + agentId)
		cmd.Env = append(cmd.Env, EnvForceTraceId+ "=" + strconv.FormatUint(traceId, 10))
		cmd.Env = append(cmd.Env, EnvForceSpanId+ "=" + strconv.FormatUint(spanId, 10))
		output, _ := cmd.CombinedOutput()

		coverageData := getCoverage(coverageFile)
		_ = coverageData

		if cmd.ProcessState.ExitCode() == 0 {
			//fmt.Println(*coverageData)
			t.SkipNow()
		} else {
			fmt.Println(string(output))
			//fmt.Println(*coverageData)
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
			traceId, _ = strconv.ParseUint(traceIdStr, 10, 64)
		}
		if spanIdStr, set := os.LookupEnv(EnvForceSpanId); set {
			spanId, _ = strconv.ParseUint(spanIdStr, 10, 64)
		}
		ok = traceId != 0 && spanId != 0
		return
	}
	return "", 0, 0, false
}

func getCoverage(coverageFile string) *coverage {
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
			endFileToken := strings.Index(line, ":")
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
