package utility

import (
	"archive/zip"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

func CreateProblemDirectory(problemId string) error {
	problemsDir := os.Getenv("PROBLEMS_DIR")
	if problemsDir == "" {
		problemsDir = "../../problems" // 預設本地開發路徑
	}

	dirPath := filepath.Join(problemsDir, problemId)
	return os.MkdirAll(dirPath, os.ModePerm)
}

func ExtractProblemFile(file multipart.File, size int64, problemId string) error {
	problemsDir := os.Getenv("PROBLEMS_DIR")
	if problemsDir == "" {
		problemsDir = "../../problems" // 預設本地開發路徑
	}

	dirPath := filepath.Join(problemsDir, problemId)

	r, err := zip.NewReader(file, size)
	if err != nil {
		return err
	}

	destClean := filepath.Clean(dirPath)

	for _, f := range r.File {
		filePath := filepath.Join(dirPath, f.Name)

		if !strings.HasPrefix(filepath.Clean(filePath), destClean+string(os.PathSeparator)) && filepath.Clean(filePath) != destClean {
			return fmt.Errorf("illegal file path in zip (possible path traversal): %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		srcFile, err := f.Open()
		if err != nil {
			dstFile.Close()
			return err
		}

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			dstFile.Close()
			srcFile.Close()
			return err
		}

		dstFile.Close()
		srcFile.Close()
	}

	return nil
}

func SaveProblemZip(file multipart.File, problemId string) error {
	problemsDir := os.Getenv("PROBLEMS_DIR")
	if problemsDir == "" {
		problemsDir = "../../problems" // 預設本地開發路徑
	}

	destPath := filepath.Join(problemsDir, problemId, "problem.zip")

	// Seek back to the beginning of the file to read it from start
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
	problemsDir := os.Getenv("PROBLEMS_DIR")
	if problemsDir == "" {
		problemsDir = "../../problems" // 預設本地開發路徑
	}

	templateDir := filepath.Join(problemsDir, problemId, "template")
	destZipPath := filepath.Join(problemsDir, problemId, "template.zip")

	info, err := os.Stat(templateDir)
	if err != nil {
		if os.IsNotExist(err) {
			// If template directory doesn't exist, we don't need to create template.zip
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}

	zipFile, err := os.Create(destZipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	err = filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = filepath.ToSlash(relPath)

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		w, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(w, file)
		return err
	})

	return err
}

func DeleteProblemDirectory(problemId string) error {
	problemsDir := os.Getenv("PROBLEMS_DIR")
	if problemsDir == "" {
		problemsDir = "../../problems" // 預設本地開發路徑
	}

	dirPath := filepath.Join(problemsDir, problemId)
	return os.RemoveAll(dirPath)
}