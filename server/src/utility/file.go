package utility

import (
	"mime/multipart"
)

type FileData struct {
	Bytes []byte
	Size  int64
}

func ToFileData(file *multipart.FileHeader) (FileData, error) {
	in, err := file.Open()
	if err != nil {
		return FileData{}, err
	}
	defer in.Close()

	bytes := make([]byte, file.Size)
	if _, err := in.Read(bytes); err != nil {
		return FileData{}, err
	}

	return FileData{
		Bytes: bytes,
		Size:  file.Size,
	}, nil
}
