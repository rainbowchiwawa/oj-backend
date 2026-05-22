package utility

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
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

// extractZip extracts a zip archive from the given reader into destDir.
// It handles Unicode filename decoding, path traversal prevention, and
// enforces size limits (MaxFileSize per file, MaxTotalSize total, MaxFiles entries).
func extractZip(src io.ReaderAt, size int64, destDir string) error {
	r, err := zip.NewReader(src, size)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}

	if len(r.File) > MaxFiles {
		return fmt.Errorf("zip contains too many files: %d (max %d)", len(r.File), MaxFiles)
	}

	destClean := filepath.Clean(destDir)
	var totalSize int64

	for _, f := range r.File {
		name := decodeZipFilename(f.Name)

		if strings.Contains(name, "..") {
			return fmt.Errorf("illegal file path in zip: %s", name)
		}

		filePath := filepath.Join(destDir, name)
		if !strings.HasPrefix(filepath.Clean(filePath), destClean+string(os.PathSeparator)) && filepath.Clean(filePath) != destClean {
			return fmt.Errorf("illegal file path in zip (possible path traversal): %s", name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}

		if err := extractZipEntry(f, filePath); err != nil {
			return err
		}

		totalSize += int64(f.UncompressedSize64)
		if totalSize > MaxTotalSize {
			return fmt.Errorf("total decompressed size exceeds limit (%d bytes)", MaxTotalSize)
		}
	}

	return nil
}

// extractZipEntry writes a single zip file entry to disk with a per-file size limit.
func extractZipEntry(f *zip.File, destPath string) error {
	dstFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}

	srcFile, err := f.Open()
	if err != nil {
		dstFile.Close()
		return err
	}

	written, err := io.Copy(dstFile, io.LimitReader(srcFile, MaxFileSize))
	dstFile.Close()
	srcFile.Close()
	if err != nil {
		return err
	}

	if written == MaxFileSize {
		return fmt.Errorf("file %s exceeds max decompressed size (%d bytes)", f.Name, MaxFileSize)
	}

	return nil
}

// compressDir compresses the contents of srcDir into a zip file at destZip.
// Symlinks are skipped to prevent reading outside the directory.
func compressDir(srcDir string, destZip string) error {
	info, err := os.Stat(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}

	zipFile, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
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

		writer, err := w.CreateHeader(header)
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

		_, err = io.Copy(writer, file)
		return err
	})
}

// zipFiles returns the parsed file list from a zip archive, useful for
// pre-validation before extraction.
func zipFiles(src io.ReaderAt, size int64) ([]*zip.File, error) {
	r, err := zip.NewReader(src, size)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}
	return r.File, nil
}
