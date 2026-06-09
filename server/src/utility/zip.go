package utility

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

const (
	MaxFileSize  = 100 << 20 // 100 MB per decompressed file
	MaxTotalSize = 500 << 20 // 500 MB total decompressed
	MaxFiles     = 1000      // max entries in zip
)

func ValidateZipStructure(file *multipart.FileHeader, required []string) error {
	return withZipFiles(file, func(files []*zip.File) error {
		requiredSet := make(map[string]struct{})
		for _, r := range required {
			requiredSet[r] = struct{}{}
		}

		for _, f := range files {
			name := decodeChineseString(f.Name)
			if _, exists := requiredSet[name]; exists {
				delete(requiredSet, name)
				continue
			}
			for _, r := range required {
				if strings.HasSuffix(r, "/") && strings.HasPrefix(name, r) {
					delete(requiredSet, name)
				}
			}
		}

		for _, r := range requiredSet {
			return fmt.Errorf("invalid problem zip format: missing %s at root", r)
		}
		return nil
	})
}

func ExtractZip(file *multipart.FileHeader, destDir string) error {
	return withZipFiles(file, func(files []*zip.File) error {
		totalSize := int64(0)
		for _, f := range files {
			srcEntryPath := decodeChineseString(f.Name)
			if strings.Contains(srcEntryPath, "..") {
				return fmt.Errorf("illegal file path in zip: %s", srcEntryPath)
			}

			destEntryPath := filepath.Join(destDir, srcEntryPath)
			if f.FileInfo().IsDir() {
				if err := os.MkdirAll(destEntryPath, os.ModePerm); err != nil {
					return err
				}
				continue
			}

			if err := os.MkdirAll(filepath.Dir(destEntryPath), os.ModePerm); err != nil {
				return err
			}

			if err := extractEntry(f, destEntryPath); err != nil {
				return err
			}

			totalSize += int64(f.UncompressedSize64)
			if totalSize > MaxTotalSize {
				return fmt.Errorf("total decompressed size exceeds limit (%d bytes)", MaxTotalSize)
			}
		}
		return nil
	})
}

func CompressZip(srcDir string, destFilePath string) error {
	info, err := os.Stat(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("srcDir is not a directory!")
	}

	out, err := os.Create(destFilePath)
	if err != nil {
		return err
	}
	defer out.Close()

	writer := zip.NewWriter(out)
	defer writer.Close()

	return filepath.WalkDir(srcDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		isLink := (entry.Type() & fs.ModeSymlink) != 0
		if isLink {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		return compressEntry(writer, path, relPath, entry)
	})
}

func extractEntry(f *zip.File, destPath string) error {
	fileSize := f.FileInfo().Size()
	if fileSize >= MaxFileSize {
		return fmt.Errorf("file %s exceeds max compressed size (%d bytes)", f.Name, MaxFileSize)
	}

	in, err := f.Open()
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	writtenSize, err := io.Copy(out, io.LimitReader(in, fileSize))
	if err != nil {
		return err
	}

	if writtenSize == MaxFileSize {
		return fmt.Errorf("file %s exceeds max decompressed size (%d bytes)", f.Name, MaxFileSize)
	}

	return nil
}

func compressEntry(writer *zip.Writer, absPath, relPath string, entry fs.DirEntry) error {
	info, err := entry.Info()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Name = filepath.ToSlash(relPath)
	if entry.IsDir() {
		header.Name += "/"
	} else {
		header.Method = zip.Deflate
	}

	out, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}

	if entry.IsDir() {
		return nil
	}

	in, err := os.Open(absPath)
	if err != nil {
		return err
	}
	defer in.Close()

	_, err = io.Copy(out, in)
	return err
}

func withZipFiles(file *multipart.FileHeader, cb func([]*zip.File) error) error {
	in, err := file.Open()
	if err != nil {
		return err
	}
	defer in.Close()

	reader, err := zip.NewReader(in, file.Size)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}

	fileCount := len(reader.File)
	if fileCount > MaxFiles {
		return fmt.Errorf("zip contains too many files: %d (max %d)", fileCount, MaxFiles)
	}

	return cb(reader.File)
}
