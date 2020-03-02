package config

type (
	ScopeConfig struct {
		Dsn             *string                `env:"SCOPE_DSN"`
		ApiKey          *string                `env:"SCOPE_APIKEY"`
		ApiEndpoint     *string                `env:"SCOPE_API_ENDPOINT"`
		Service         *string                `yaml:"service" env:"SCOPE_SERVICE" default:"default"`
		Repository      *string                `yaml:"repository" env:"SCOPE_REPOSITORY"`
		CommitSha       *string                `yaml:"commit_sha" env:"SCOPE_COMMIT_SHA"`
		Branch          *string                `yaml:"branch" env:"SCOPE_BRANCH"`
		SourceRoot      *string                `yaml:"source_root" env:"SCOPE_SOURCE_ROOT"`
		Logger          LoggerConfig           `yaml:"logger"`
		Metadata        map[string]interface{} `yaml:"metadata" env:"SCOPE_METADATA"`
		Configuration   []string               `yaml:"configuration" env:"SCOPE_CONFIGURATION" default:"platform.name, platform.architecture, go.version"`
		TestingMode     *bool                  `yaml:"testing_mode" env:"SCOPE_TESTING_MODE" default:"false"`
		Instrumentation InstrumentationConfig  `yaml:"instrumentation"`
		Tracer          TracerConfig           `yaml:"tracer"`
		Debug           *bool                  `env:"SCOPE_DEBUG" default:"false"`
		ConfigPath      *string
		LoadError       error
	}
	LoggerConfig struct {
		Root *string `yaml:"root" env:"SCOPE_LOGGER_ROOT, SCOPE_LOG_ROOT_PATH"`
	}
	InstrumentationConfig struct {
		DiffSummary     *bool                                `yaml:"diff_summary" env:"SCOPE_INSTRUMENTATION_DIFF_SUMMARY" default:"true"`
		TestsFrameworks InstrumentationTestsFrameworksConfig `yaml:"tests_frameworks"`
		DB              InstrumentationDatabaseConfig        `yaml:"db"`
		Http            InstrumentationHttpConfig            `yaml:"http"`
		Logger          InstrumentationLoggerConfig          `yaml:"logger"`
	}
	InstrumentationTestsFrameworksConfig struct {
		FailRetries *int  `yaml:"fail_retries" env:"SCOPE_INSTRUMENTATION_TESTS_FRAMEWORKS_FAIL_RETRIES" default:"0"`
		PanicAsFail *bool `yaml:"panic_as_fail" env:"SCOPE_INSTRUMENTATION_TESTS_FRAMEWORKS_PANIC_AS_FAIL" default:"false"`
	}
	InstrumentationDatabaseConfig struct {
		StatementValues *bool `yaml:"statement_values" env:"SCOPE_INSTRUMENTATION_DB_STATEMENT_VALUES" default:"false"`
		Stacktrace *bool `yaml:"stacktrace" env:"SCOPE_INSTRUMENTATION_DB_STACKTRACE" default:"false"`
	}
	InstrumentationHttpConfig struct {
		Client   *bool    `yaml:"client" env:"SCOPE_INSTRUMENTATION_HTTP_CLIENT" default:"true"`
		Server   *bool    `yaml:"server" env:"SCOPE_INSTRUMENTATION_HTTP_SERVER" default:"true"`
		Payloads *bool    `yaml:"payloads" env:"SCOPE_INSTRUMENTATION_HTTP_PAYLOADS" default:"false"`
		Stacktrace *bool `yaml:"stacktrace" env:"SCOPE_INSTRUMENTATION_HTTP_STACKTRACE" default:"false"`
		Headers  []string `yaml:"headers" env:"SCOPE_INSTRUMENTATION_HTTP_HEADERS"`
	}
	InstrumentationLoggerConfig struct {
		StandardLogger *bool `yaml:"standard_logger" env:"SCOPE_INSTRUMENTATION_LOGGER_STANDARD_LOGGER" default:"true"`
		StandardOutput *bool `yaml:"standard_output" env:"SCOPE_INSTRUMENTATION_LOGGER_STANDARD_OUTPUT" default:"false"`
		StandardError  *bool `yaml:"standard_error" env:"SCOPE_INSTRUMENTATION_LOGGER_STANDARD_ERROR" default:"false"`
	}
	TracerConfig struct {
		Global     *bool                  `yaml:"global" env:"SCOPE_TRACER_GLOBAL, SCOPE_SET_GLOBAL_TRACER" default:"false"`
		Dispatcher TracerDispatcherConfig `yaml:"dispatcher"`
	}
	TracerDispatcherConfig struct {
		HealthCheckFrequency           *int                         `yaml:"healthcheck_frecuency" env:"SCOPE_TRACER_DISPATCHER_HEALTHCHECK_FRECUENCY" default:"1000"`
		HealthCheckFrequencyInTestMode *int                         `yaml:"healthcheck_frecuency_in_testmode" env:"SCOPE_TRACER_DISPATCHER_HEALTHCHECK_FRECUENCY_IN_TESTMODE" default:"60000"`
		ConcurrencyLevel               *int                         `yaml:"concurrency_level" env:"SCOPE_TRACER_DISPATCHER_CONCURRENCY_LEVEL" default:"1"`
		Spans                          TracerDispatcherSpansConfig  `yaml:"spans"`
		Events                         TracerDispatcherEventsConfig `yaml:"events"`
	}
	TracerDispatcherSpansConfig struct {
		MaxPayloadSize *int `yaml:"max_payload_size" env:"SCOPE_TRACER_DISPATCHER_SPANS_MAX_PAYLOAD_SIZE" default:"1000"`
	}
	TracerDispatcherEventsConfig struct {
		MaxPayloadSize *int `yaml:"max_payload_size" env:"SCOPE_TRACER_DISPATCHER_EVENTS_MAX_PAYLOAD_SIZE" default:"1000"`
	}
)
