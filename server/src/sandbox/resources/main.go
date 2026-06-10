package resources

import (
	"oj/server/utility"
	"path/filepath"
)

var problemBasePath string
var submissionBasePath string

func Init() {
	problemBasePath = filepath.Join(utility.EnvData.BasePath, "problems")
	submissionBasePath = filepath.Join(utility.EnvData.BasePath, "submissions")
}
