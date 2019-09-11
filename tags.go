package scopeagent

const (
	AgentType    = "agent.type"
	AgentID      = "agent.id"
	AgentVersion = "agent.version"

	PlatformName         = "platform.name"
	PlatformArchitecture = "platform.architecture"
	ProcessArchitecture  = "architecture"

	CurrentFolder = "current.folder"
	Hostname      = "hostname"

	GoVersion = "go.version"

	Service    = "service"
	Repository = "repository"
	Commit     = "commit"
	Branch     = "branch"
	SourceRoot = "source.root"
	Diff       = "diff"

	CI            = "ci.in_ci"
	CIProvider    = "ci.provider"
	CIBuildId     = "ci.build_id"
	CIBuildNumber = "ci.build_number"
	CIBuildUrl    = "ci.build_url"

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
)
