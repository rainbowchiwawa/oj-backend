package sandbox

import (
	"fmt"
	"log"
	"mime/multipart"
	"oj/server/utility"
	"os"
	"path/filepath"
)

var problemsDir string

func init() {
	problemsDir = filepath.Join(utility.EnvData.BasePath + "/problems")
}

func SaveProblemFile(problemId string, isNew bool, file *multipart.FileHeader, deleteProblem func() error) error {
	success := false
	defer func() {
		if success {
			if !isNew {
				cleanupProblemBackup(problemId)
			}
			return
		}
		if cleanErr := deleteProblemDirectory(problemId); cleanErr != nil {
			log.Printf("rollback: failed to delete problem directory %s: %v", problemId, cleanErr)
		}
		if isNew {
			if cleanErr := deleteProblem(); cleanErr != nil {
				log.Printf("rollback: failed to delete problem record %s: %v", problemId, cleanErr)
			}
		} else {
			if cleanErr := restoreProblemDirectory(problemId); cleanErr != nil {
				log.Printf("rollback: failed to restore old problem directory %s: %v", problemId, cleanErr)
			}
		}
	}()

	if !isNew {
		if err := backupProblemDirectory(problemId); err != nil {
			return fmt.Errorf("cannot backup old problem directory")
		}
	}

	if err := createProblemDirectory(problemId); err != nil {
		return fmt.Errorf("cannot create problem directory")
	}

	if err := extractProblemFile(file, problemId); err != nil {
		return fmt.Errorf("cannot extract problem file: " + err.Error())
	}

	if err := saveProblemZip(file, problemId); err != nil {
		return fmt.Errorf("cannot save problem zip")
	}

	if err := createProblemTemplateZip(problemId); err != nil {
		return fmt.Errorf("cannot create template zip")
	}
	success = true
	return nil
}

func DeleteProblemFile(problemId string, deleteProblem func() error) error {
	if err := backupProblemDirectory(problemId); err != nil {
		return fmt.Errorf("cannot backup problem directory: " + err.Error())
	}

	if err := deleteProblem(); err != nil {
		if restoreErr := restoreProblemDirectory(problemId); restoreErr != nil {
			return fmt.Errorf("rollback: failed to restore problem directory %s: %v", problemId, restoreErr)
		}
		return fmt.Errorf("cannot delete problem from database: " + err.Error())
	}

	cleanupProblemBackup(problemId)
	return nil
}

func GetProblemFilePath(problemId string, filename string) string {
	return filepath.Join(problemsDir, problemId, filename)
}

func createProblemDirectory(problemId string) error {
	dirPath := filepath.Join(problemsDir, problemId)
	return os.MkdirAll(dirPath, os.ModePerm)
}

func extractProblemFile(file *multipart.FileHeader, problemId string) error {
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

func saveProblemZip(file *multipart.FileHeader, problemId string) error {
	destPath := filepath.Join(problemsDir, problemId, "problem.zip")
	data, err := utility.ToFileData(file)
	if err != nil {
		return err
	}
	return os.WriteFile(destPath, data.Bytes, 0666)
}

func createProblemTemplateZip(problemId string) error {
	templateDir := filepath.Join(problemsDir, problemId, "template")
	destZipPath := filepath.Join(problemsDir, problemId, "template.zip")
	return utility.CompressZip(templateDir, destZipPath)
}

func deleteProblemDirectory(problemId string) error {
	dirPath := filepath.Join(problemsDir, problemId)
	return os.RemoveAll(dirPath)
}

func backupProblemDirectory(problemId string) error {
	src := filepath.Join(problemsDir, problemId)
	dst := filepath.Join(problemsDir, problemId+".bak")
	os.RemoveAll(dst)
	return os.Rename(src, dst)
}

func restoreProblemDirectory(problemId string) error {
	src := filepath.Join(problemsDir, problemId+".bak")
	dst := filepath.Join(problemsDir, problemId)
	os.RemoveAll(dst)
	return os.Rename(src, dst)
}

func cleanupProblemBackup(problemId string) {
	dst := filepath.Join(problemsDir, problemId+".bak")
	os.RemoveAll(dst)
}
