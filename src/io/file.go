package io

// The parts of dart:io's File that painting's FileImage uses: path,
// length and readAsBytes. File is an interface (Dart code implements it,
// e.g. package:file's in-memory files); NewFile returns dart:io's own,
// *LocalFile.

import (
	"os"

	"github.com/jonathanperis/flutter-go/src/async"
)

// File is a reference to a file on a file system.
type File interface {
	// Path is the file's path.
	Path() string
	// Length is the file's length in bytes.
	Length() async.Future[int]
	// ReadAsBytes reads the whole file.
	ReadAsBytes() async.Future[[]byte]
	// String is Dart's toString ("File: '<path>'").
	String() string
}

// LocalFile is dart:io's File (its runtime type is File in Dart).
type LocalFile struct{ path string }

// NewFile is File(path).
func NewFile(path string) *LocalFile { return &LocalFile{path: path} }

func (f *LocalFile) Path() string { return f.path }

// Length is the file's length (the future fails if it can't be read).
func (f *LocalFile) Length() async.Future[int] {
	return async.NewFuture(func() int {
		st, err := os.Stat(f.path)
		if err != nil {
			panic(err)
		}
		return int(st.Size())
	})
}

// ReadAsBytes reads the file (the future fails if it can't be read).
func (f *LocalFile) ReadAsBytes() async.Future[[]byte] {
	return async.NewFuture(func() []byte {
		b, err := os.ReadFile(f.path)
		if err != nil {
			panic(err)
		}
		return b
	})
}

func (f *LocalFile) String() string { return "File: '" + f.path + "'" }
