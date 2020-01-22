package testing

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	_ "unsafe"

	"github.com/google/uuid"
	"go.undefinedlabs.com/scopeagent/instrumentation"
)

type (
	coverage struct {
		Type    string         `json:"type" msgpack:"type"`
		Version string         `json:"version" msgpack:"version"`
		Uuid    string         `json:"uuid" msgpack:"uuid"`
		Files   []fileCoverage `json:"files" msgpack:"files"`
	}
	fileCoverage struct {
		Filename   string  `json:"filename" msgpack:"filename"`
		Boundaries [][]int `json:"boundaries" msgpack:"boundaries"`
	}
	pkg struct {
		ImportPath string
		Dir        string
		Error      *struct {
			Err string
		}
	}
)

//go:linkname cover testing.cover
var (
	cover         testing.Cover
	counters      map[string][]uint32
	countersMutex sync.Mutex
	filePathData  map[string]string
)

// Initialize coverage
func InitCoverage() {
	if filePathData == nil {
		var files []string
		for key := range cover.Blocks {
			files = append(files, key)
		}
		pkgData, err := findPkgs(files)
		if err != nil {
			pkgData = map[string]*pkg{}
			instrumentation.Logger().Printf("Coverage error: %v", err)
		}
		filePathData = map[string]string{}
		for key := range cover.Blocks {
			filePath, err := findFile(pkgData, key)
			if err != nil {
				instrumentation.Logger().Printf("Coverage error: %v", err)
			} else {
				filePathData[key] = filePath
			}
		}
	}
}

// Clean the counters for a new coverage session
func startCoverage() {
	countersMutex.Lock()
	defer countersMutex.Unlock()

	if counters == nil {
		counters = map[string][]uint32{}
		InitCoverage()
	}
	for name, counts := range cover.Counters {
		counters[name] = make([]uint32, len(counts))
		for i := range counts {
			counters[name][i] = counts[i]
			counts[i] = 0
		}
	}
}

// Get the counters values and extract the coverage info
func endCoverage() *coverage {
	countersMutex.Lock()
	defer countersMutex.Unlock()

	fileMap := map[string][][]int{}
	var count uint32
	for name, counts := range cover.Counters {
		if file, ok := filePathData[name]; ok {
			blocks := cover.Blocks[name]
			blockStack := make([]*testing.CoverBlock, 0)
			for i := range counts {
				count = atomic.LoadUint32(&counts[i])
				atomic.StoreUint32(&counts[i], counters[name][i]+count)
				if count > 0 {
					curBlock := blocks[i]
					var prvBlock *testing.CoverBlock
					blockStackLen := len(blockStack)
					if blockStackLen > 0 {
						prvBlock = blockStack[blockStackLen-1]
					}

					if prvBlock == nil {
						fileMap[file] = append(fileMap[file], []int{
							int(curBlock.Line0), int(curBlock.Col0), int(count),
						})
						blockStack = append(blockStack, &curBlock)
					} else if contains(prvBlock, &curBlock) {
						pBoundCol := int(curBlock.Col0)
						cBoundCol := int(curBlock.Col0)
						if pBoundCol > 0 {
							pBoundCol--
						} else {
							cBoundCol++
						}
						fileMap[file] = append(fileMap[file], []int{
							int(curBlock.Line0), pBoundCol, -1,
						})
						fileMap[file] = append(fileMap[file], []int{
							int(curBlock.Line0), cBoundCol, int(count),
						})
						blockStack = append(blockStack, &curBlock)
					} else {
						pBoundCol := int(prvBlock.Col1)
						cBoundCol := int(curBlock.Col0)
						if pBoundCol > 0 {
							pBoundCol--
						} else {
							cBoundCol++
						}
						fileMap[file] = append(fileMap[file], []int{
							int(prvBlock.Line1), pBoundCol, -1,
						})
						fileMap[file] = append(fileMap[file], []int{
							int(curBlock.Line0), cBoundCol, int(count),
						})
						blockStack[blockStackLen-1] = &curBlock
					}
				}
			}

			if len(blockStack) > 0 {
				prvBlock := blockStack[len(blockStack)-1]
				fileMap[file] = append(fileMap[file], []int{
					int(prvBlock.Line1), int(prvBlock.Col1), -1,
				})
			}
		}

	}
	files := make([]fileCoverage, 0)
	for key, value := range fileMap {
		files = append(files, fileCoverage{
			Filename:   key,
			Boundaries: value,
		})
	}
	uuidValue, _ := uuid.NewRandom()
	coverageData := &coverage{
		Type:    "com.undefinedlabs.uccf",
		Version: "0.2.0",
		Uuid:    uuidValue.String(),
		Files:   files,
	}
	return coverageData
}

func contains(outter, inner *testing.CoverBlock) bool {
	if outter != nil && inner != nil {
		return outter.Line0 < inner.Line0 &&
			outter.Col0 < inner.Col0 &&
			outter.Line1 > inner.Line1 &&
			outter.Col1 > inner.Col1
	}
	return false
}

// Functions to find the absolute path from coverage data.
// Extracted from: https://github.com/golang/go/blob/master/src/cmd/cover/func.go
func findPkgs(fileNames []string) (map[string]*pkg, error) {
	// Run go list to find the location of every package we care about.
	pkgs := make(map[string]*pkg)
	var list []string
	for _, filename := range fileNames {
		if strings.HasPrefix(filename, ".") || filepath.IsAbs(filename) {
			// Relative or absolute path.
			continue
		}
		pkg := path.Dir(filename)
		if _, ok := pkgs[pkg]; !ok {
			pkgs[pkg] = nil
			list = append(list, pkg)
		}
	}

	if len(list) == 0 {
		return pkgs, nil
	}

	// Note: usually run as "go tool cover" in which case $GOROOT is set,
	// in which case runtime.GOROOT() does exactly what we want.
	goTool := filepath.Join(runtime.GOROOT(), "bin/go")
	cmd := exec.Command(goTool, append([]string{"list", "-e", "-json"}, list...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cannot run go list: %v\n%s", err, stderr.Bytes())
	}
	dec := json.NewDecoder(bytes.NewReader(stdout))
	for {
		var pkg pkg
		err := dec.Decode(&pkg)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decoding go list json: %v", err)
		}
		pkgs[pkg.ImportPath] = &pkg
	}
	return pkgs, nil
}

// findFile finds the location of the named file in GOROOT, GOPATH etc.
func findFile(pkgs map[string]*pkg, file string) (string, error) {
	if strings.HasPrefix(file, ".") || filepath.IsAbs(file) {
		// Relative or absolute path.
		return file, nil
	}
	pkg := pkgs[path.Dir(file)]
	if pkg != nil {
		if pkg.Dir != "" {
			return filepath.Join(pkg.Dir, path.Base(file)), nil
		}
		if pkg.Error != nil {
			return "", errors.New(pkg.Error.Err)
		}
	}
	return "", fmt.Errorf("did not find package for %s in go list output", file)
}
