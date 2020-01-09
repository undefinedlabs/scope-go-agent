package testing

import (
	"github.com/google/uuid"
	"sync/atomic"
	"testing"
	_ "unsafe"
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
)

//go:linkname cover testing.cover
var (
	cover    testing.Cover
	counters map[string][]uint32
)

// Clean the counters for a new coverage session
func (test *Test) startCoverage() {
	if counters == nil {
		counters = map[string][]uint32{}
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
func (test *Test) endCoverage() *coverage {
	fileMap := map[string][][]int{}
	var active, total int64
	var count uint32
	for name, counts := range cover.Counters {
		blocks := cover.Blocks[name]
		for i := range counts {
			stmts := int64(blocks[i].Stmts)
			total += stmts
			count = atomic.LoadUint32(&counts[i])
			atomic.StoreUint32(&counts[i], counters[name][i]+count)
			if count > 0 {
				active += stmts
				fileMap[name] = append(fileMap[name], []int{
					int(blocks[i].Line0), int(blocks[i].Col0), int(count),
				})
				fileMap[name] = append(fileMap[name], []int{
					int(blocks[i].Line1), int(blocks[i].Col1), -1,
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
