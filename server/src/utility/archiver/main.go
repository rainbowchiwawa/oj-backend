package archiver

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileHeader struct {
	Name    string
	Mode    os.FileMode
	Size    int64
	ModTime time.Time
	IsDir   bool
}

type Writer interface {
	WriteFile(header FileHeader, content io.Reader) error
	Close() error
}

type Reader interface {
	Next() (FileHeader, io.Reader, error)
}

func CompressFiles(w Writer, basePath string, files []string) error {
	for _, f := range files {
		abs := filepath.Join(basePath, f)
		info, err := os.Stat(abs)
		if err != nil {
			return err
		}

		if info.IsDir() {
			err = filepath.Walk(abs, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}

				rel, err := filepath.Rel(basePath, path)
				if err != nil || rel == "." {
					return err
				}

				return compressEntry(w, path, rel, info)
			})
			if err != nil {
				return err
			}
		} else {
			if err := compressEntry(w, abs, f, info); err != nil {
				return err
			}
		}
	}
	return nil
}

func CompressDir(w Writer, srcDir string) error {
	return filepath.Walk(srcDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil || rel == "." {
			return err
		}

		return compressEntry(w, path, rel, info)
	})
}

func compressEntry(w Writer, abs, rel string, info fs.FileInfo) error {
	h := FileHeader{
		Name:    filepath.ToSlash(rel),
		Mode:    info.Mode(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}
	if info.IsDir() {
		return w.WriteFile(h, nil)
	}
	if !info.Mode().IsRegular() {
		return nil
	}

	in, err := os.Open(abs)
	if err != nil {
		return err
	}
	defer in.Close()
	return w.WriteFile(h, in)
}

func ExtractTo(r Reader, destDir string, options ...[]string) error {
	requiredSet := make(map[string]struct{})
	var required []string
	if len(options) > 0 {
		required = options[0]
	} else {
		required = []string{}
	}
	for _, r := range required {
		requiredSet[r] = struct{}{}
	}

	for {
		header, content, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if _, exists := requiredSet[header.Name]; exists {
			delete(requiredSet, header.Name)
			continue
		}
		for _, r := range required {
			if strings.HasSuffix(r, "/") && strings.HasPrefix(header.Name, r) {
				delete(requiredSet, header.Name)
			}
		}
		if strings.Contains(header.Name, "..") {
			return fmt.Errorf("illegal header name %s", header.Name)
		}

		destEntryPath := filepath.Join(destDir, filepath.FromSlash(header.Name))
		if header.IsDir {
			if err := os.MkdirAll(destEntryPath, header.Mode.Perm()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destEntryPath), 0666); err != nil {
			return err
		}

		out, err := os.OpenFile(destEntryPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, header.Mode.Perm())
		if err != nil {
			return err
		}
		defer out.Close()

		if _, err := io.Copy(out, content); err != nil {
			return err
		}
	}

	for r := range requiredSet {
		return fmt.Errorf("invalid problem zip format: missing %s at root", r)
	}
	return nil
}
