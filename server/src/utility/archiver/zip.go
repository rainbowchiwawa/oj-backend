package archiver

import (
	"archive/zip"
	"io"
)

type ZipWriter struct {
	writer *zip.Writer
}

func NewZipWriter(w io.Writer) *ZipWriter {
	return &ZipWriter{writer: zip.NewWriter(w)}
}

func (z *ZipWriter) WriteFile(h FileHeader, content io.Reader) error {
	zh := &zip.FileHeader{
		Name:     h.Name,
		Modified: h.ModTime,
		Method:   zip.Deflate,
	}
	zh.SetMode(h.Mode)
	if h.IsDir {
		zh.Name += "/"
	}
	out, err := z.writer.CreateHeader(zh)
	if err != nil {
		return err
	}
	if !h.IsDir && content != nil {
		_, err = io.Copy(out, content)
	}
	return err
}

func (z *ZipWriter) Close() error {
	return z.writer.Close()
}

type ZipReader struct {
	reader *zip.Reader
	idx    int
	cur    io.ReadCloser
}

func NewZipReader(r io.ReaderAt, size int64) (*ZipReader, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, err
	}
	return &ZipReader{reader: zr}, nil
}

func (z *ZipReader) Next() (FileHeader, io.Reader, error) {
	if z.cur != nil {
		z.cur.Close()
		z.cur = nil
	}
	if z.idx >= len(z.reader.File) {
		return FileHeader{}, nil, io.EOF
	}
	entry := z.reader.File[z.idx]
	z.idx++

	info := entry.FileInfo()
	h := FileHeader{
		Name:    decodeChineseString(entry.Name),
		Mode:    info.Mode(),
		Size:    info.Size(),
		ModTime: entry.Modified,
		IsDir:   info.IsDir(),
	}
	if h.IsDir {
		return h, nil, nil
	}

	in, err := entry.Open()
	if err != nil {
		return h, nil, err
	}

	z.cur = in
	return h, in, nil
}
