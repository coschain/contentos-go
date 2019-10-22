package setup

const (
	DefaultValueSignal = "d"
	EmptyLine          = ""
	Positive           = "yes"
	Negative           = "no"
)

// read type
const (
	IsBp     = "IsBp"
	YesOrNo  = "YesOrNo"
	NodeName = "NodeName"
	ChainId  = "ChainId"
	BpName   = "BpName"
	PriKey   = "PriKey"
	SeedList = "SeedList"
	LogLevel = "LogLevel"
	DataDir  = "DataDir"

	StartNode = "StartNode"
)

var (
	InitNewConfig = true
	NodeIsBp      = false
	StartNodeNow  = false
)