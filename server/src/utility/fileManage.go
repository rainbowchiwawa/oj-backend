package utility

import (
	"archive/zip"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
)

var problemsDir string

func init() {
	problemsDir = filepath.Join(EnvData.BasePath + "/problems")
}

func GetProblemFilePath(problemId string, filename string) string {
	return filepath.Join(problemsDir, problemId, filename)
}

func CreateProblemDirectory(problemId string) error {
	dirPath := filepath.Join(problemsDir, problemId)
	return os.MkdirAll(dirPath, os.ModePerm)
}

func ExtractProblemFile(file multipart.File, size int64, problemId string) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	files, err := zipFiles(file, size)
	if err != nil {
		return err
	}
	if err := validateProblemZipStructure(files); err != nil {
		return err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	destDir := filepath.Join(problemsDir, problemId)
	return extractZip(file, size, destDir)
}

func SaveProblemZip(file multipart.File, problemId string) error {
	destPath := filepath.Join(problemsDir, problemId, "problem.zip")

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	return err
}

func CreateProblemTemplateZip(problemId string) error {
	templateDir := filepath.Join(problemsDir, problemId, "template")
	destZipPath := filepath.Join(problemsDir, problemId, "template.zip")
	return compressDir(templateDir, destZipPath)
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

func WriteFileToPath(file *multipart.FileHeader, path string) error {
	inputFile, err := file.Open()
	if err != nil {
		return err
	}
	defer inputFile.Close()

	bytes := make([]byte, file.Size)
	if _, err := inputFile.Read(bytes); err != nil {
		return err
	}

	outputFile, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	if _, err := outputFile.Write(bytes); err != nil {
		return err
	}

	return nil
}

func validateProblemZipStructure(files []*zip.File) error {
	type requiredEntry struct {
		name  string
		isDir bool
		found bool
	}
	required := []requiredEntry{
		{name: "CMakeLists.txt", isDir: false},
		{name: "cmake", isDir: true},
		{name: "solution", isDir: true},
		{name: "spec", isDir: true},
		{name: "template", isDir: true},
	}

	for _, f := range files {
		normName := filepath.ToSlash(filepath.Clean(decodeZipFilename(f.Name)))
		for i := range required {
			if required[i].found {
				continue
			}
			if required[i].isDir {
				if normName == required[i].name || strings.HasPrefix(normName, required[i].name+"/") {
					required[i].found = true
				}
			} else if normName == required[i].name {
				required[i].found = true
			}
		}
	}

	for _, req := range required {
		if !req.found {
			kind := "file"
			if req.isDir {
				kind = "directory"
			}
			return fmt.Errorf("invalid problem zip format: missing %s %s at root", kind, req.name)
		}
	}
	return nil
}

func decodeZipFilename(name string) string {
	if utf8.ValidString(name) {
		return name
	}
	decoder := traditionalchinese.Big5.NewDecoder()
	if decoded, err := decoder.String(name); err == nil && !strings.Contains(decoded, "\uFFFD") {
		return decoded
	}
	decoderGBK := simplifiedchinese.GBK.NewDecoder()
	if decoded, err := decoderGBK.String(name); err == nil && !strings.Contains(decoded, "\uFFFD") {
		return decoded
	}
	return name
}
