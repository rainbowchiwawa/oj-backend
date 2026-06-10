package archiver

import (
	"archive/tar"
	"io"
	"os"
)

type TarWriter struct {
	writer *tar.Writer
}

func NewTarWriter(w io.Writer) *TarWriter {
	return &TarWriter{writer: tar.NewWriter(w)}
}

func (t *TarWriter) WriteFile(header FileHeader, content io.Reader) error {
	entry := &tar.Header{
		Name:     header.Name,
		Mode:     int64(header.Mode.Perm()),
		Size:     header.Size,
		ModTime:  header.ModTime,
		Typeflag: tar.TypeReg,
	}
	if header.IsDir {
		entry.Typeflag = tar.TypeDir
		entry.Name += "/"
		entry.Size = 0
	}
	if err := t.writer.WriteHeader(entry); err != nil {
		return err
	}
	if !header.IsDir && content != nil {
		_, err := io.Copy(t.writer, content)
		return err
	}
	return nil
}

func (t *TarWriter) Close() error {
	return t.writer.Close()
}

type TarReader struct {
	reader *tar.Reader
}

func NewTarReader(r io.Reader) *TarReader {
	return &TarReader{reader: tar.NewReader(r)}
}

func (t *TarReader) Next() (FileHeader, io.Reader, error) {
	entry, err := t.reader.Next()
	if err != nil {
		return FileHeader{}, nil, err
	}
	return FileHeader{
		Name:    decodeChineseString(entry.Name),
		Mode:    os.FileMode(entry.Mode),
		Size:    entry.Size,
		ModTime: entry.ModTime,
		IsDir:   entry.Typeflag == tar.TypeDir,
	}, t.reader, nil
}
