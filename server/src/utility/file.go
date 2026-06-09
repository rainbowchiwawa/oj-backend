package utility

import (
	"mime/multipart"
	"os"
)

type FileData struct {
	Bytes []byte
	Size  int64
}

func MultipartToFileData(file *multipart.FileHeader) (FileData, error) {
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

func CopyFile(srcPath string, destPath string) error {
	bytes, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	if err = os.WriteFile(destPath, bytes, os.ModePerm); err != nil {
		return err
	}
	return nil
}
