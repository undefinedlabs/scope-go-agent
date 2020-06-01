package instrumentation

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/opentracing/opentracing-go"
)

var (
	tracer       opentracing.Tracer = opentracing.NoopTracer{}
	logger                          = log.New(ioutil.Discard, "", 0)
	sourceRoot                      = ""
	remoteConfig                    = map[string]interface{}{}

	m sync.RWMutex
)

func SetTracer(t opentracing.Tracer) {
	m.Lock()
	defer m.Unlock()

	tracer = t
}

func Tracer() opentracing.Tracer {
	m.RLock()
	defer m.RUnlock()

	return tracer
}

func SetLogger(l *log.Logger) {
	m.Lock()
	defer m.Unlock()

	logger = l
}

func Logger() *log.Logger {
	m.RLock()
	defer m.RUnlock()

	return logger
}

func SetSourceRoot(root string) {
	m.Lock()
	defer m.Unlock()
	// In windows the debug symbols and source root can be in different cases
	if runtime.GOOS == "windows" {
		root = strings.ToLower(root)
	}
	sourceRoot = root
}

func GetSourceRoot() string {
	m.RLock()
	defer m.RUnlock()

	return sourceRoot
}

func SetRemoteConfiguration(config map[string]interface{}) {
	m.Lock()
	defer m.Unlock()

	remoteConfig = config
}

func GetRemoteConfiguration() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()

	return remoteConfig
}

//go:noinline
func GetCallerInsideSourceRoot(skip int) (pc uintptr, file string, line int, ok bool) {
	isWindows := runtime.GOOS == "windows"
	pcs := make([]uintptr, 64)
	count := runtime.Callers(skip+2, pcs)
	pcs = pcs[:count]
	frames := runtime.CallersFrames(pcs)
	for {
		frame, more := frames.Next()
		file := filepath.Clean(frame.File)
		dir := filepath.Dir(file)
		// In windows the debug symbols and source root can be in different cases
		if isWindows {
			dir = strings.ToLower(dir)
		}
		if strings.Index(dir, sourceRoot) != -1 {
			return frame.PC, file, frame.Line, true
		}
		if !more {
			break
		}
	}
	return
}
