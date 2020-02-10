package env

import (
	"fmt"
	"os"
	"strconv"
)

type EnvVar struct {
	key      string
	value    string
	hasValue bool
}

var (
	SCOPE_DEBUG                   = newEnvVar("SCOPE_DEBUG")
	SCOPE_DSN                     = newEnvVar("SCOPE_DSN")
	SCOPE_APIKEY                  = newEnvVar("SCOPE_APIKEY")
	SCOPE_API_ENDPOINT            = newEnvVar("SCOPE_API_ENDPOINT")
	SCOPE_TESTING_MODE            = newEnvVar("SCOPE_TESTING_MODE")
	SCOPE_SET_GLOBAL_TRACER       = newEnvVar("SCOPE_SET_GLOBAL_TRACER")
	SCOPE_TESTING_FAIL_RETRIES    = newEnvVar("SCOPE_TESTING_FAIL_RETRIES")
	SCOPE_TESTING_PANIC_AS_FAIL   = newEnvVar("SCOPE_TESTING_PANIC_AS_FAIL")
	SCOPE_LOG_ROOT_PATH           = newEnvVar("SCOPE_LOG_ROOT_PATH")
	SCOPE_REPOSITORY              = newEnvVar("SCOPE_REPOSITORY")
	SCOPE_COMMIT_SHA              = newEnvVar("SCOPE_COMMIT_SHA")
	SCOPE_SOURCE_ROOT             = newEnvVar("SCOPE_SOURCE_ROOT")
	SCOPE_SERVICE                 = newEnvVar("SCOPE_SERVICE")
	SCOPE_DISABLE_MONKEY_PATCHING = newEnvVar("SCOPE_DISABLE_MONKEY_PATCHING")
)

func newEnvVar(key string) EnvVar {
	value, hasValue := os.LookupEnv(key)
	return EnvVar{
		key:      key,
		value:    value,
		hasValue: hasValue,
	}
}

func (e *EnvVar) AsBool(fallback bool) bool {
	if !e.hasValue {
		return fallback
	}
	value, err := strconv.ParseBool(e.value)
	if err != nil {
		panic(fmt.Sprintf("unable to parse %s - should be 'true' or 'false'", e.key))
	}
	return value
}

func (e *EnvVar) AsInt(fallback int) int {
	if !e.hasValue {
		return fallback
	}
	value, err := strconv.ParseInt(e.value, 0, 0)
	if err != nil {
		panic(fmt.Sprintf("unable to parse %s - does not seem to be an int", e.key))
	}
	return int(value)
}

func (e *EnvVar) AsString(fallback string) string {
	if !e.hasValue {
		return fallback
	}
	return e.value
}

func (e *EnvVar) AsTuple() (string, bool) {
	return e.value, e.hasValue
}

func IfFalse(expression bool, envVar EnvVar, fallback bool) bool {
	if expression {
		return true
	}
	return envVar.AsBool(fallback)
}

func IfIntZero(defInt int, envVar EnvVar, fallback int) int {
	if defInt != 0 {
		return defInt
	}
	return envVar.AsInt(fallback)
}

func AddStringToMapIfEmpty(source map[string]interface{}, key string, envVar EnvVar, fallback string) {
	if val, ok := source[key]; !ok || val == "" {
		source[key] = envVar.AsString(fallback)
	}
}
