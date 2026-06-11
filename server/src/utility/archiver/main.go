package archiver

import (
	"bytes"
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

type PathTestFunc func(string) bool

func isPathMatch(name string, path string) bool {
	return name == path || strings.HasSuffix(name, "/") && strings.HasPrefix(path, name)
}

type CompressEntry struct {
	Name     string
	Test     PathTestFunc
	Compress CompressFunc
}
type CompressFunc func() error

func CompressDir(w Writer, srcDir string, cb func(CompressEntry) error) error {
	return filepath.Walk(srcDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil || rel == "." {
			return err
		}

		if cb != nil {
			return cb(CompressEntry{
				Name:     filepath.ToSlash(rel),
				Test:     func(name string) bool { return isPathMatch(name, filepath.ToSlash(rel)) },
				Compress: func() error { return compressEntry(w, path, rel, info) },
			})
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

type ExtractEntry struct {
	Name    string
	Test    PathTestFunc
	Extract ExtractFunc
	Read    ReadFunc
}
type ExtractFunc func() error
type ReadFunc func() ([]byte, error)

func ExtractTo(r Reader, destDir, root string, cb func(ExtractEntry) error) error {
	for {
		header, content, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if strings.Contains(header.Name, "..") {
			return fmt.Errorf("illegal header name %s", header.Name)
		}

		rel, err := filepath.Rel(root, header.Name)
		if err != nil {
			return err
		}

		destEntryPath := filepath.Join(destDir, filepath.FromSlash(rel))
		if cb != nil {
			if err = cb(ExtractEntry{
				Name:    rel,
				Test:    func(name string) bool { return isPathMatch(name, rel) },
				Extract: func() error { return extractEntry(destEntryPath, header, content) },
				Read: func() ([]byte, error) {
					var buf bytes.Buffer
					if _, err := io.Copy(&buf, content); err != nil {
						return nil, err
					}
					return buf.Bytes(), nil
				},
			}); err != nil {
				return err
			}
			continue
		}

		if err = extractEntry(destEntryPath, header, content); err != nil {
			return err
		}
	}
	return nil
}

func extractEntry(destEntryPath string, header FileHeader, content io.Reader) error {
	if header.IsDir {
		if err := os.MkdirAll(destEntryPath, header.Mode.Perm()); err != nil {
			return err
		}
		return nil
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
	return nil
}
