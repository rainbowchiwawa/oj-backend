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

	if err = submissionManager.saveZip(file); err != nil {
		return err
	}
	return nil
}

func (s SubmissionManager) GetBasePath() string {
	return filepath.Join(utility.EnvData.SubmissionBasePath, s.Id)
}

func (s SubmissionManager) GetZipPath() string {
	return filepath.Join(utility.EnvData.SubmissionBasePath, s.Id+".zip")
}

func (s SubmissionManager) GetChildPath(childPath SubmissionChildPath) string {
	return filepath.Join(s.GetBasePath(), string(childPath))
}

func (s SubmissionManager) saveZip(file *multipart.FileHeader) error {
	zipPath := s.GetZipPath()
	data, err := utility.MultipartToFileData(file)
	if err != nil {
		return err
	}

	return os.WriteFile(zipPath, data.Bytes, 0666)
}

func (s SubmissionManager) ExtractZip() error {
	zipPath := s.GetZipPath()
	extractPath := s.GetChildPath(SubmissionExtractDir)
	if err := os.MkdirAll(extractPath, os.ModePerm); err != nil {
		return err
	}

	in, err := os.Open(zipPath)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	zr, err := archiver.NewZipReader(in, info.Size())
	if err != nil {
		return err
	}
	return archiver.ExtractTo(zr, extractPath, "", nil)
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

	os.Remove(filepath.Join(extractDir, "case.h"))

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
