package models

type Testcase struct {
	Name string            `json:"name"`
	Args map[string]string `json:"args"`
}
