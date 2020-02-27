package config

type (
	ScopeConfig struct {
		Dsn             *string                `env:"SCOPE_DSN"`
		ApiKey          *string                `env:"SCOPE_APIKEY"`
		ApiEndpoint     *string                `env:"SCOPE_API_ENDPOINT" default:"https://app.scope.dev"`
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
		Debug           *bool                  `env:"SCOPE_DEBUG" default:"false"`
	}
	LoggerConfig struct {
		Root *string `yaml:"root" env:"SCOPE_LOGGER_ROOT, SCOPE_LOG_ROOT_PATH"`
	}
	InstrumentationConfig struct {
		Enabled         *bool                                `yaml:"enabled" env:"SCOPE_INSTRUMENTATION_ENABLED" default:"true"`
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
	}
	InstrumentationHttpConfig struct {
		Client   *bool    `yaml:"client" env:"SCOPE_INSTRUMENTATION_HTTP_CLIENT" default:"true"`
		Server   *bool    `yaml:"server" env:"SCOPE_INSTRUMENTATION_HTTP_SERVER" default:"true"`
		Payloads *bool    `yaml:"payloads" env:"SCOPE_INSTRUMENTATION_HTTP_PAYLOADS" default:"false"`
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
		HealthCheckFrequency           *int                         `yaml:"healthcheck_frecuency" env:"SCOPE_TRACER_DISPATCHER_HEALTHCHECK_FRECUENCY" default:"1"`
		HealthCheckFrequencyInTestMode *int                         `yaml:"healthcheck_frecuency_in_testmode" env:"SCOPE_TRACER_DISPATCHER_HEALTHCHECK_FRECUENCY_IN_TESTMODE" default:"60"`
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

/*
service: 'service-name'                                           #SCOPE_SERVICE
repository: 'https://github.com/undefinedlabs/scope-docs.git'     #SCOPE_REPOSITORY
commit_sha: '974c3566eb8e221d130db86a7ce1f99703fe2e69'            #SCOPE_COMMIT_SHA
branch: 'master'                                                  #SCOPE_BRANCH
source_root: '/home/user/projects/scope-docs'                     #SCOPE_SOURCE_ROOT
logger:
  level: 'trace'                                                  #SCOPE_LOGGER_LEVEL
  root: '/home/user/projects/scope/log'                           #SCOPE_LOGGER_ROOT
code_path:
  enabled: true                                                   #SCOPE_CODE_PATH_ENABLED
  base_packages: 'com.undefinedlabs.scope'                        #SCOPE_CODE_PATH_BASE_PACKAGES
  debug: true                                                     #SCOPE_CODE_PATH_DEBUG
metadata:                                                         #SCOPE_METADATA
  sample.key1: $SAMPLE_VAR1
  sample.key2: $SAMPLE_VAR2
  sample.key3: sampleValue3
configuration:                                                    #SCOPE_CONFIGURATION
  - sample.key1
  - sample.key2
  - sample.key3
testing_mode: true                                                #SCOPE_TESTING_MODE
instrumentation:
  enabled: true                                                   #SCOPE_INSTRUMENTATION_ENABLED
  diff_summary: true                                              #SCOPE_INSTRUMENTATION_DIFF_SUMMARY
  tests_frameworks:
    fail_retries: 5                                               #SCOPE_INSTRUMENTATION_TESTS_FRAMEWORKS_FAIL_RETRIES
    libraries:
      mstest: true                                                #SCOPE_INSTRUMENTATION_TESTS_FRAMEWORKS_LIBRARIES_MSTEST
      nunit: true                                                 #SCOPE_INSTRUMENTATION_TESTS_FRAMEWORKS_LIBRARIES_NUNIT
      xunit: true                                                 #SCOPE_INSTRUMENTATION_TESTS_FRAMEWORKS_LIBRARIES_XUNIT
  db:
    execution_plan: true                                          #SCOPE_INSTRUMENTATION_DB_EXECUTION_PLAN
    execution_plan_threshold: 0                                   #SCOPE_INSTRUMENTATION_DB_EXECUTION_PLAN_THRESHOLD
    statement_values: true                                        #SCOPE_INSTRUMENTATION_DB_STATEMENT_VALUES
    libraries:
      entityframework_core: true                                  #SCOPE_INSTRUMENTATION_DB_LIBRARIES_ENTITYFRAMEWORK_CORE
      redis: true                                                 #SCOPE_INSTRUMENTATION_DB_LIBRARIES_REDIS
      sqlserver: true                                             #SCOPE_INSTRUMENTATION_DB_LIBRARIES_SQLSERVER
      mysql: true                                                 #SCOPE_INSTRUMENTATION_DB_LIBRARIES_MYSQL
      postgres: true                                              #SCOPE_INSTRUMENTATION_DB_LIBRARIES_POSTGRES
      sqlite: true                                                #SCOPE_INSTRUMENTATION_DB_LIBRARIES_SQLITE
      mongodb: true                                               #SCOPE_INSTRUMENTATION_DB_LIBRARIES_MONGODB
  http:
    client: true                                                  #SCOPE_INSTRUMENTATION_HTTP_CLIENT
    server: true                                                  #SCOPE_INSTRUMENTATION_HTTP_SERVER
    libraries:
      aspnet_core: true                                           #SCOPE_INSTRUMENTATION_HTTP_LIBRARIES_ASPNET_CORE
    payloads: true                                                #SCOPE_INSTRUMENTATION_HTTP_PAYLOADS
    headers:                                                      #SCOPE_INSTRUMENTATION_HTTP_HEADERS
      - Authorization
      - My-Header-One
      - My-Header-Two
  logger:
    standard_trace: true                                          #SCOPE_INSTRUMENTATION_LOGGER_STANDARD_TRACE
    libraries:
      microsoft_logging: true                                     #SCOPE_INSTRUMENTATION_LOGGER_LIBRARIES_MICROSOFT_LOGGING
      serilog: true                                               #SCOPE_INSTRUMENTATION_LOGGER_LIBRARIES_SERILOG
      nlog: true                                                  #SCOPE_INSTRUMENTATION_LOGGER_LIBRARIES_NLOG
      log4net: true                                               #SCOPE_INSTRUMENTATION_LOGGER_LIBRARIES_LOG4NET
tracer:
  global: true                                                    #SCOPE_TRACER_GLOBAL
  dispatcher:
    healthcheck_frecuency: 10000                                  #SCOPE_TRACER_DISPATCHER_HEALTHCHECK_FRECUENCY
    healthcheck_frecuency_in_testmode: 1000                       #SCOPE_TRACER_DISPATCHER_HEALTHCHECK_FRECUENCY_IN_TESTMODE
    concurrency_level: 5                                          #SCOPE_TRACER_DISPATCHER_CONCURRENCY_LEVEL
    close_timeout: 30000                                          #SCOPE_TRACER_DISPATCHER_CLOSE_TIMEOUT
    spans:
      max_buffer_size: 1000                                       #SCOPE_TRACER_DISPATCHER_SPANS_MAX_BUFFER_SIZE
      max_payload_size: -1                                        #SCOPE_TRACER_DISPATCHER_SPANS_MAX_PAYLOAD_SIZE
    events:
      max_buffer_size: 1000                                       #SCOPE_TRACER_DISPATCHER_EVENTS_MAX_BUFFER_SIZE
      max_payload_size: -1                                        #SCOPE_TRACER_DISPATCHER_EVENTS_MAX_PAYLOAD_SIZE

*/
