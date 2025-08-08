package parsers

type StageName = string
type Stage struct {
	Testcases  []Testcase        `yaml:"testcases"`
	Before     []StageName       `yaml:"before,omitempty"`
	Args       map[string]string `yaml:"args,omitempty"`
	PreScript  Content           `yaml:"preScript"`
	PostScript Content           `yaml:"postScript"`
	Score      int64             `yaml:"score"`
}
