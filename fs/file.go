package fs

import (
	"os"
	"path/filepath"
	"strings"

	"ddbt/jinja/ast"
)

type FileType string

const (
	UnknownFile FileType = "UNKNOWN"
	ModelFile            = "model"
	MacroFile            = "macro"
	TestFile             = "test"
)

type File struct {
	Type       FileType
	Name       string
	Path       string
	SyntaxTree ast.AST

	PrereadFileContents string // Used for testing
}

func newFile(path string, file os.FileInfo, fileType FileType) *File {
	return &File{
		Type: fileType,
		Name: strings.TrimSuffix(filepath.Base(path), ".sql"),
		Path: path,
	}
}
