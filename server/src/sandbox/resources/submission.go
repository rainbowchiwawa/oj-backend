package resources

import (
	"mime/multipart"
	"oj/server/utility"
	"oj/server/utility/archiver"
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

func (s SubmissionManager) GetChildPath(childPath SubmissionChildPath) string {
	return filepath.Join(s.GetBasePath(), string(childPath))
}

func (s SubmissionManager) extractAndSave(file *multipart.FileHeader) error {
	zipPath := s.GetChildPath(SubmissionZip)
	extractPath := s.GetChildPath(SubmissionExtractDir)
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
	return archiver.ExtractTo(zr, extractPath)
}

func (s SubmissionManager) CopyTestFiles(p ProblemManager) (err error) {
	extractDir := s.GetChildPath(SubmissionExtractDir)
	err = utility.CopyFile(p.getChildPath(ProblemCMakeLists), filepath.Join(extractDir, "CMakeLists.txt"))
	if err != nil {
		println("failed to copy cmakelists", err.Error())
		return
	}

	err = utility.CopyFile(p.getChildPath(ProblemEntryPoint), filepath.Join(extractDir, "entrypoint.cpp"))
	if err != nil {
		println("failed to copy entrypoint", err.Error())
		return
	}

	err = utility.CopyFile(p.getChildPath(ProblemTestHeader), filepath.Join(extractDir, "test.h"))
	if err != nil {
		println("failed to copy test header", err.Error())
		return
	}

	specDir := filepath.Join(s.GetBasePath(), "spec")
	err = os.CopyFS(specDir, os.DirFS(p.getChildPath(ProblemSpec)))
	if err != nil {
		println("failed to copy spec dir", err.Error())
		return
	}

	return nil
}

func (s SubmissionManager) ClearFiles() error {
	return os.RemoveAll(s.GetBasePath())
}
