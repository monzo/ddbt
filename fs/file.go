package fs

import (
	"os"
	"strings"
)

type FileType int
const (
	UnknownFile FileType = iota
	ModelFile
	MacroFile
	TestFile
)

type File struct {
	Type FileType
	Name string
	Path string
}

func newFile(path string, file os.FileInfo, fileType FileType) *File {
	return &File{
		Type: fileType,
		Name: strings.TrimSuffix(file.Name(), ".sql"),
		Path: path,
	}
}