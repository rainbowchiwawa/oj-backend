package resources

import (
	"mime/multipart"
	"oj/server/utility"
	"os"
	"path/filepath"
)

type SubmissionChildPath string

const (
	SubmissionZip        SubmissionChildPath = "source.zip"
	SubmissionExtractDir SubmissionChildPath = "src"
)

type SubmissionManager struct {
	Id string
}

var submissionBasePath string

func init() {
	submissionBasePath = filepath.Join(utility.EnvData.BasePath + "/submissions")
}

func SaveUploadedSubmission(submissionId string, file *multipart.FileHeader, deleteSubmission func() error) (err error) {
	submissionManager := SubmissionManager{Id: submissionId}
	defer func() {
		if err != nil {
			deleteSubmission()
		}
	}()

	if err = submissionManager.extractAndSave(file); err != nil {
		return err
	}
	return nil
}

func (s SubmissionManager) GetBasePath() string {
	return filepath.Join(submissionBasePath, s.Id)
}

func (s SubmissionManager) getChildPath(childPath SubmissionChildPath) string {
	return filepath.Join(s.GetBasePath(), string(childPath))
}

func (s SubmissionManager) extractAndSave(file *multipart.FileHeader) error {
	zipPath := s.getChildPath(SubmissionZip)
	extractPath := s.getChildPath(SubmissionExtractDir)
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

	return utility.ExtractZip(file, extractPath)
}

func (s SubmissionManager) CopyTestFiles(p ProblemManager) (err error) {

	extractDir := s.getChildPath(SubmissionExtractDir)
	err = utility.CopyFile(p.getChildPath(ProblemEntryPoint), filepath.Join(extractDir, "entrypoint.cpp"))
	if err != nil {
		return
	}

	err = utility.CopyFile(p.getChildPath(ProblemTestHeader), filepath.Join(extractDir, "test.h"))
	if err != nil {
		return
	}

	rootDir := s.GetBasePath()
	return os.CopyFS(rootDir, os.DirFS(p.getChildPath(ProblemSpec)))
}

func (s SubmissionManager) ClearFiles() error {
	return os.RemoveAll(s.GetBasePath())
}
