package runtimeinfo

var OperatorVersion = "0.1.0"
var GitCommit = "dev"
var BuildDate = "unknown"

type Info struct {
	OperatorVersion string `json:"operatorVersion"`
	GitCommit       string `json:"gitCommit"`
	BuildDate       string `json:"buildDate"`
}
