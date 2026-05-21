package utility

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

// Zip extraction safety limits.
const (
	MaxFileSize  = 100 << 20 // 100 MB per decompressed file
	MaxTotalSize = 500 << 20 // 500 MB total decompressed
	MaxFiles     = 1000      // max entries in zip
)

// problemsDir is the base directory for problem data, initialized once.
var problemsDir string

func init() {
	problemsDir = os.Getenv("PROBLEMS_DIR")
	if problemsDir == "" {
		problemsDir = "../../problems" // 預設本地開發路徑
	}
}

// GetProblemFilePath returns the absolute path for a file within a problem directory.
func GetProblemFilePath(problemId string, filename string) string {
	return filepath.Join(problemsDir, problemId, filename)
}

// ValidateZipMagic checks that the first 4 bytes of the file match the zip
// magic number (PK\x03\x04). The file offset is reset to the start afterwards.
func ValidateZipMagic(file multipart.File) error {
	magic := make([]byte, 4)
	if _, err := file.Read(magic); err != nil {
		return errors.New("uploaded file is not a valid zip")
	}
	if string(magic) != "PK\x03\x04" {
		return errors.New("uploaded file is not a valid zip")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek after magic check: %w", err)
	}
	return nil
}

func CreateProblemDirectory(problemId string) error {
	dirPath := filepath.Join(problemsDir, problemId)
	return os.MkdirAll(dirPath, os.ModePerm)
}

func ExtractProblemFile(file multipart.File, size int64, problemId string) error {
	// #1 — Each consumer owns its own offset; seek to start explicitly.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	dirPath := filepath.Join(problemsDir, problemId)

	r, err := zip.NewReader(file, size)
	if err != nil {
		return err
	}

	// #10 — Limit file count to prevent inode/fd exhaustion.
	if len(r.File) > MaxFiles {
		return fmt.Errorf("zip contains too many files: %d (max %d)", len(r.File), MaxFiles)
	}

	destClean := filepath.Clean(dirPath)
	var totalSize int64

	for _, f := range r.File {
		// #9 — Reject paths containing ".." before any further processing.
		if strings.Contains(f.Name, "..") {
			return fmt.Errorf("illegal file path in zip: %s", f.Name)
		}

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

		// #3 — Limit per-file decompressed size via LimitReader.
		written, err := io.Copy(dstFile, io.LimitReader(srcFile, MaxFileSize))
		dstFile.Close()
		srcFile.Close()
		if err != nil {
			return err
		}

		// Check if the file was truncated by LimitReader (meaning it exceeded the limit).
		if written == MaxFileSize {
			return fmt.Errorf("file %s exceeds max decompressed size (%d bytes)", f.Name, MaxFileSize)
		}

		totalSize += written
		if totalSize > MaxTotalSize {
			return fmt.Errorf("total decompressed size exceeds limit (%d bytes)", MaxTotalSize)
		}
	}

	return nil
}

func SaveProblemZip(file multipart.File, problemId string) error {
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

	// #11 — Use WalkDir and skip symlinks to prevent reading outside the directory.
	err = filepath.WalkDir(templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip symlinks to prevent following links to outside the directory.
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = filepath.ToSlash(relPath)

		if d.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		w, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if d.IsDir() {
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
	dirPath := filepath.Join(problemsDir, problemId)
	return os.RemoveAll(dirPath)
}

// BackupProblemDirectory renames the problem directory to a .bak suffix.
// Any leftover backup from a previous failed attempt is removed first.
func BackupProblemDirectory(problemId string) error {
	src := filepath.Join(problemsDir, problemId)
	dst := filepath.Join(problemsDir, problemId+".bak")
	os.RemoveAll(dst) // remove stale backup if any
	return os.Rename(src, dst)
}

// RestoreProblemDirectory restores the .bak backup to the original location.
// Any partially-created new directory is removed first.
func RestoreProblemDirectory(problemId string) error {
	src := filepath.Join(problemsDir, problemId+".bak")
	dst := filepath.Join(problemsDir, problemId)
	os.RemoveAll(dst) // remove partial new directory
	return os.Rename(src, dst)
}

// CleanupProblemBackup removes the .bak backup directory after a successful operation.
func CleanupProblemBackup(problemId string) {
	dst := filepath.Join(problemsDir, problemId+".bak")
	os.RemoveAll(dst)
}