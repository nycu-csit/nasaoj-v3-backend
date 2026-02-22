package types

type (
	RestyClient  string
	AccessToken  string
	RefreshToken string
	Privileges   string
)

type ToolInputTypeType = int

const (
	ToolInputTypeString ToolInputTypeType = iota
	ToolInputTypeNumeric
	ToolInputTypeBoolean
)
