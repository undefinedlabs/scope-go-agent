package tags

import (
	"fmt"
	"reflect"
)

const (
	AgentType    = "agent.type"
	AgentID      = "agent.id"
	AgentVersion = "agent.version"

	PlatformName         = "platform.name"
	PlatformArchitecture = "platform.architecture"
	ProcessArchitecture  = "architecture"

	CurrentFolder = "current.folder"
	Hostname      = "hostname"

	InContainer = "incontainer"
	GoVersion   = "go.version"

	Service    = "service"
	Repository = "repository"
	Commit     = "commit"
	Branch     = "branch"
	SourceRoot = "source.root"
	Diff       = "diff"

	Capabilities               = "capabilities"
	Capabilities_CodePath      = "code.path"
	Capabilities_ProcessEnd    = "process.end"
	Capabilities_RunnerRetries = "runner.retries"
	Capabilities_RunnerCache   = "runner.cache"

	CI            = "ci.in_ci"
	CIProvider    = "ci.provider"
	CIBuildId     = "ci.build_id"
	CIBuildNumber = "ci.build_number"
	CIBuildUrl    = "ci.build_url"

	Dependencies = "dependencies"

	EventType      = "event"
	EventSource    = "source"
	EventMessage   = "message"
	EventStack     = "stack"
	EventException = "exception"

	EventTestFailure = "test_failure"
	EventTestSkip    = "test_skip"

	LogEvent      = "log"
	LogEventLevel = "log.level"

	LogLevel_INFO    = "INFO"
	LogLevel_WARNING = "WARNING"
	LogLevel_ERROR   = "ERROR"
	LogLevel_DEBUG   = "DEBUG"
	LogLevel_VERBOSE = "VERBOSE"

	TestStatus_FAIL  = "FAIL"
	TestStatus_PASS  = "PASS"
	TestStatus_SKIP  = "SKIP"
	TestStatus_CACHE = "CACHE"

	TestingMode = "testing"

	ConfigurationKeys = "configuration.keys"

	Coverage = "test.coverage"
)

func GetValidValue(value interface{}) (interface{}, bool) {
	if value == nil {
		return nil, false
	}
	if vs, ok := value.(fmt.Stringer); ok {
		return vs.String(), true
	}
	rValue := reflect.ValueOf(value)
	for {
		rKind := rValue.Kind()
		if rKind == reflect.Ptr {
			rValue = rValue.Elem()
			continue
		}
		if (rKind < 1 || rKind > 16) && rKind != reflect.String {
			return fmt.Sprint(value), true
		}
		break
	}
	return value, false
}
