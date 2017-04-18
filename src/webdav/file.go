package main

import (
	"os"
)

type File struct {
	file *os.File
}

func (f File) Close() error {
	return f.file.Close()
}

func (f File) Read(p []byte) (n int, err error) {
	return f.file.Read(p)
}

func (f File) Seek(offset int64, whence int) (int64, error) {
	return f.file.Seek(offset, whence)
}

func (f File) Write(p []byte) (n int, err error) {
	return f.file.Write(p)
}

func (f File) Readdir(count int) ([]os.FileInfo, error) {
	return f.file.Readdir(count)
}

func (f File) Stat() (os.FileInfo, error) {
	return f.file.Stat()
}
