package scopeagent

const (
	AgentType    = "agent.type"
	AgentID      = "agent.id"
	AgentVersion = "agent.version"

	Service    = "service"
	Repository = "repository"
	Commit     = "commit"
	SourceRoot = "source.root"

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


	LogEvent			= "log"
	LogEventLevel		= "log.level"

	LogLevel_INFO		= "INFO"
	LogLevel_WARNING 	= "WARNING"
	LogLevel_ERROR		= "ERROR"
	LogLevel_DEBUG		= "DEBUG"
	LogLevel_VERBOSE	= "VERBOSE"
)
