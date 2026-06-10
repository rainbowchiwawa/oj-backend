package resources

import (
	"fmt"
	"log"
	"mime/multipart"
	"oj/server/parser"
	"oj/server/utility"
	"oj/server/utility/archiver"
	"os"
	"path/filepath"
)

type ProblemFilePath string

const (
	ProblemZip         ProblemFilePath = "problem.zip"
	ProblemTemplateZip ProblemFilePath = "template.zip"
	ProblemPublicDir   ProblemFilePath = "public"
	ProblemExtractDir  ProblemFilePath = "src"
	ProblemSpec        ProblemFilePath = "src/spec"
	ProblemAnswer      ProblemFilePath = "src/online-judge/result.xml"
	ProblemCMakeLists  ProblemFilePath = "src/CMakeLists.txt"
	ProblemSettings    ProblemFilePath = "src/settings.yaml"
	ProblemTemplateDir ProblemFilePath = "src/template"
	ProblemEntryPoint  ProblemFilePath = "src/template/entrypoint.cpp"
	ProblemTestHeader  ProblemFilePath = "src/template/test.h"
)

type ProblemManager struct {
	Id string
}

func SaveUploadedProblem(
	problemId string,
	isNew bool,
	file *multipart.FileHeader,
	updateConfig func(*parser.TestResults, *parser.ProblemSettings) error,
	deleteProblem func() error,
) (err error) {
	problemManager := ProblemManager{Id: problemId}
	defer func() {
		if err == nil {
			if !isNew {
				problemManager.cleanupBackup()
			}
			return
		}
		if cleanErr := problemManager.delete(); cleanErr != nil {
			log.Printf("rollback: failed to delete problem directory %s: %v", problemId, cleanErr)
		}
		if isNew {
			if cleanErr := deleteProblem(); cleanErr != nil {
				log.Printf("rollback: failed to delete problem record %s: %v", problemId, cleanErr)
			}
		} else {
			if cleanErr := problemManager.restore(); cleanErr != nil {
				log.Printf("rollback: failed to restore old problem directory %s: %v", problemId, cleanErr)
			}
		}
	}()

	if !isNew {
		if err = problemManager.backup(); err != nil {
			return fmt.Errorf("cannot backup old problem directory")
		}
	}

	if err = problemManager.extractAndSave(file); err != nil {
		return fmt.Errorf("cannot extract problem file: " + err.Error())
	}

	bytes, err := os.ReadFile(problemManager.getChildPath(ProblemSettings))
	if err != nil {
		return err
	}

	settings, err := parser.ParseProblemSettings(bytes)
	if err != nil {
		return err
	}

	if err = problemManager.createTemplateZip(settings.Public); err != nil {
		return fmt.Errorf("cannot create template zip")
	}

	resultBytes, err := os.ReadFile(problemManager.getChildPath(ProblemAnswer))
	if err != nil {
		return err
	}
	result, err := parser.ParseTestResults(resultBytes)
	if err != nil {
		return err
	}

	if err = updateConfig(result, settings); err != nil {
		return err
	}

	return nil
}

func DeleteUploadedProblem(problemId string, deleteProblem func() error) error {
	problem := ProblemManager{Id: problemId}
	if err := problem.backup(); err != nil {
		return fmt.Errorf("cannot backup problem directory: " + err.Error())
	}

	if err := deleteProblem(); err != nil {
		if restoreErr := problem.restore(); restoreErr != nil {
			return fmt.Errorf("rollback: failed to restore problem directory %s: %v", problemId, restoreErr)
		}
		return fmt.Errorf("cannot delete problem from database: " + err.Error())
	}

	problem.cleanupBackup()
	return nil
}

func GetProblemFilePath(problemId string, childPath ProblemFilePath) (string, bool, error) {
	problem := ProblemManager{Id: problemId}
	fullPath := problem.getChildPath(childPath)
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return fullPath, true, nil
}

func (p ProblemManager) getBasePath() string {
	return filepath.Join(problemBasePath, p.Id)
}

func (p ProblemManager) getChildPath(relPath ProblemFilePath) string {
	return filepath.Join(p.getBasePath(), string(relPath))
}

func (p ProblemManager) getBackupPath() string {
	return filepath.Join(problemBasePath, p.Id+".bak")
}

func (p ProblemManager) extractAndSave(file *multipart.FileHeader) error {
	zipPath := p.getChildPath(ProblemZip)
	extractPath := p.getChildPath(ProblemExtractDir)
	if err := os.MkdirAll(extractPath, os.ModePerm); err != nil {
		return err
	}

	data, err := utility.MultipartToFileData(file)
	if err != nil {
		return err
	}

	if err = os.WriteFile(zipPath, data.Bytes, 0666); err != nil {
		return err
	}

	in, err := file.Open()
	if err != nil {
		return err
	}
	defer in.Close()

	zr, err := archiver.NewZipReader(in, file.Size)
	if err != nil {
		return err
	}

	return archiver.ExtractTo(zr, extractPath, []string{
		"CMakeLists.txt",
		"settings.yaml",
		"spec/",
		"online-judge/result.xml",
		"template/entrypoint.cpp",
		"template/test.h",
	})
}

func (p ProblemManager) createTemplateZip(public []parser.ProblemPublicPair) error {
	for _, s := range public {
		srcPath := filepath.Join(p.getChildPath(ProblemExtractDir), s.Source)
		destPath := filepath.Join(p.getChildPath(ProblemPublicDir), s.Target)
		if err := utility.CopyFile(srcPath, destPath); err != nil {
			return err
		}
	}
	templateDir := p.getChildPath(ProblemPublicDir)
	destZipPath := p.getChildPath(ProblemTemplateZip)

	out, err := os.Create(destZipPath)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := archiver.NewZipWriter(out)
	defer zw.Close()

	return archiver.CompressDir(zw, templateDir)
}

func (p ProblemManager) delete() error {
	dirPath := p.getBasePath()
	return os.RemoveAll(dirPath)
}

func (p ProblemManager) backup() error {
	src := p.getBasePath()
	dst := p.getBackupPath()
	os.MkdirAll(src, os.ModePerm)
	os.RemoveAll(dst)
	return os.Rename(src, dst)
}

func (p ProblemManager) restore() error {
	src := p.getBackupPath()
	dst := p.getBasePath()
	os.RemoveAll(dst)
	return os.Rename(src, dst)
}

func (p ProblemManager) cleanupBackup() {
	dst := p.getBackupPath()
	os.RemoveAll(dst)
}
