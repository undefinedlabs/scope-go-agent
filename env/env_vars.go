package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type EnvironmentVar struct {
	key      string
	value    string
	hasValue bool
}

var (
	ScopeDsn                   = newEnvVar("SCOPE_DSN")
	ScopeApiKey                = newEnvVar("SCOPE_APIKEY")
	ScopeApiEndpoint           = newEnvVar("SCOPE_API_ENDPOINT")
	ScopeService               = newEnvVar("SCOPE_SERVICE")
	ScopeRepository            = newEnvVar("SCOPE_REPOSITORY")
	ScopeCommitSha             = newEnvVar("SCOPE_COMMIT_SHA")
	ScopeBranch                = newEnvVar("SCOPE_BRANCH")
	ScopeSourceRoot            = newEnvVar("SCOPE_SOURCE_ROOT")
	ScopeLoggerRoot            = newEnvVar("SCOPE_LOGGER_ROOT", "SCOPE_LOG_ROOT_PATH")
	ScopeDisableMonkeyPatching = newEnvVar("SCOPE_DISABLE_MONKEY_PATCHING")
	ScopeDebug                 = newEnvVar("SCOPE_DEBUG")
	ScopeTracerGlobal          = newEnvVar("SCOPE_TRACER_GLOBAL", "SCOPE_SET_GLOBAL_TRACER")
	ScopeTestingMode           = newEnvVar("SCOPE_TESTING_MODE")
	ScopeTestingFailRetries    = newEnvVar("SCOPE_TESTING_FAIL_RETRIES")
	ScopeTestingPanicAsFail    = newEnvVar("SCOPE_TESTING_PANIC_AS_FAIL")
	ScopeConfiguration         = newEnvVar("SCOPE_CONFIGURATION")
	ScopeMetadata              = newEnvVar("SCOPE_METADATA")
)

func newEnvVar(keys ...string) EnvironmentVar {
	var eVar EnvironmentVar
	for _, key := range keys {
		value, hasValue := os.LookupEnv(key)
		eVar = EnvironmentVar{
			key:      key,
			value:    value,
			hasValue: hasValue,
		}
		if hasValue {
			break
		}
	}
	return eVar
}

func (e *EnvironmentVar) AsBool(fallback bool) bool {
	if !e.hasValue {
		return fallback
	}
	value, err := strconv.ParseBool(e.value)
	if err != nil {
		panic(fmt.Sprintf("unable to parse %s - should be 'true' or 'false'", e.key))
	}
	return value
}

func (e *EnvironmentVar) AsInt(fallback int) int {
	if !e.hasValue {
		return fallback
	}
	value, err := strconv.ParseInt(e.value, 0, 0)
	if err != nil {
		panic(fmt.Sprintf("unable to parse %s - does not seem to be an int", e.key))
	}
	return int(value)
}

func (e *EnvironmentVar) AsString(fallback string) string {
	if !e.hasValue {
		return fallback
	}
	return e.value
}

func (e *EnvironmentVar) AsSlice(fallback []string) []string {
	if !e.hasValue {
		return fallback
	}
	val := strings.Split(e.value, ",")
	for i := range val {
		val[i] = strings.TrimSpace(val[i])
	}
	return val
}

func (e *EnvironmentVar) AsMap(fallback map[string]interface{}) map[string]interface{} {
	if !e.hasValue {
		return fallback
	}
	valItems := e.AsSlice([]string{})
	val := map[string]interface{}{}
	for _, item := range valItems {
		itemArr := strings.Split(item, "=")
		if len(itemArr) == 2 {
			itemValue := itemArr[1]
			if len(itemValue) > 0 && itemValue[0] == '$' {
				itemValue = os.Getenv(itemValue[1:])
			}
			val[itemArr[0]] = itemValue
		}
	}
	return val
}

func (e *EnvironmentVar) AsTuple() (string, bool) {
	return e.value, e.hasValue
}

func GetIfFalse(expression bool, envVar EnvironmentVar, fallback bool) bool {
	if expression {
		return true
	}
	return envVar.AsBool(fallback)
}

func GetIfIntZero(defInt int, envVar EnvironmentVar, fallback int) int {
	if defInt != 0 {
		return defInt
	}
	return envVar.AsInt(fallback)
}

func AddStringToMapIfEmpty(source map[string]interface{}, key string, envVar EnvironmentVar, fallback string) {
	if val, ok := source[key]; !ok || val == "" {
		source[key] = envVar.AsString(fallback)
	}
}

func AddSliceToMapIfEmpty(source map[string]interface{}, key string, envVar EnvironmentVar, fallback []string) {
	if _, ok := source[key]; ok {
		return
	} else if val := envVar.AsSlice(fallback); val != nil {
		source[key] = val
	}
}

func MergeMapToMap(source map[string]interface{}, envVar EnvironmentVar, fallback map[string]interface{}) {
	if val := envVar.AsMap(fallback); val == nil {
		return
	} else {
		for k, v := range val {
			if _, ok := source[k]; !ok {
				source[k] = v
			}
		}
	}
}
