package sandbox

import (
	"mime/multipart"
	"oj/server/utility"
	"os"
	"path/filepath"
)

var problemsDir string

func init() {
	problemsDir = filepath.Join(utility.EnvData.BasePath + "/problems")
}

func GetProblemFilePath(problemId string, filename string) string {
	return filepath.Join(problemsDir, problemId, filename)
}

func CreateProblemDirectory(problemId string) error {
	dirPath := filepath.Join(problemsDir, problemId)
	return os.MkdirAll(dirPath, os.ModePerm)
}

func ExtractProblemFile(file *multipart.FileHeader, problemId string) error {
	err := utility.ValidateZipStructure(file, []string{
		"CMakeLists.txt",
		"settings.yaml",
		"cmake/",
		"solution/",
		"spec/",
		"template/",
	})
	if err != nil {
		return err
	}

	destDir := filepath.Join(problemsDir, problemId)
	return utility.ExtractZip(file, destDir)
}

func SaveProblemZip(data utility.FileData, problemId string) error {
	destPath := filepath.Join(problemsDir, problemId, "problem.zip")
	err := os.WriteFile(destPath, data.Bytes, 0666)
	return err
}

func CreateProblemTemplateZip(problemId string) error {
	templateDir := filepath.Join(problemsDir, problemId, "template")
	destZipPath := filepath.Join(problemsDir, problemId, "template.zip")
	return utility.CompressZip(templateDir, destZipPath)
}

func DeleteProblemDirectory(problemId string) error {
	dirPath := filepath.Join(problemsDir, problemId)
	return os.RemoveAll(dirPath)
}

func BackupProblemDirectory(problemId string) error {
	src := filepath.Join(problemsDir, problemId)
	dst := filepath.Join(problemsDir, problemId+".bak")
	os.RemoveAll(dst)
	return os.Rename(src, dst)
}

func RestoreProblemDirectory(problemId string) error {
	src := filepath.Join(problemsDir, problemId+".bak")
	dst := filepath.Join(problemsDir, problemId)
	os.RemoveAll(dst)
	return os.Rename(src, dst)
}

func CleanupProblemBackup(problemId string) {
	dst := filepath.Join(problemsDir, problemId+".bak")
	os.RemoveAll(dst)
}
