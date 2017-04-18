package main

import (
	"bufio"
	"fmt"
	"github.com/hashicorp/golang-lru"
	"golang.org/x/net/webdav"
	"io"
	"io/ioutil"
	"os"
)

type FileSystem struct {
	root     string
	cache    *Cache
	readonly bool
	log      io.Writer
	debuglog io.Writer
}

func NewFileSystem(root string, cacheLength int, prefetch int64) (*FileSystem, error) {
	fs := new(FileSystem)
	fs.root = root
	if cacheLength > 0 {
		c, err := lru.New(cacheLength)
		if err != nil {
			return nil, err
		}
		cache := Cache{Cache: *c}
		cache.prefetch = prefetch
		fs.cache = &cache
		fs.log = ioutil.Discard      // Default to discarding all logs
		fs.debuglog = ioutil.Discard // Default to discarding all debug logs
	}

	go fs.DebugPrinter()
	return fs, nil
}

func (fs *FileSystem) SetLog(out io.Writer) {
	fs.log = out
}

func (fs *FileSystem) SetDebugLog(out io.Writer) {
	fs.debuglog = out
}

func (fs *FileSystem) DebugPrinter() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if scanner.Text() == "c" {
			fmt.Fprintf(fs.debuglog, "Cache length: ", fs.cache.Len())
			for _, key := range fs.cache.Keys() {
				v, ok := fs.cache.Get(key)
				if !ok {
					continue
				}
				obj, ok := v.(*CacheObject)
				if !ok {
					panic("Failed to cast cacheobject")
				}
				obj.RLock()
				fmt.Fprintf(fs.debuglog, "Cache Content: %v, %v, %v\n", key, len(*obj.object), obj.name)
				obj.RUnlock()
			}
		}
	}
}

func (fs *FileSystem) Mkdir(name string, perm os.FileMode) error {
	fmt.Fprintf(fs.log, "Mkdir: %v\n", name)
	if fs.readonly {
		return os.ErrPermission
	}
	_name := fmt.Sprintf("%v/%v", fs.root, name)
	return os.Mkdir(_name, perm)
}

func (fs *FileSystem) OpenFile(name string, flag int, perm os.FileMode) (webdav.File, error) {
	_name := fmt.Sprintf("%v/%v", fs.root, name)
	fmt.Fprintf(fs.log, "Opening file: %v, flag: %v\n", name, flag)
	if fs.readonly {
		flag = os.O_RDONLY
		perm = os.FileMode(0444)
	}
	if fs.cache != nil {
		return NewCachingFile(_name, flag, perm, fs)
	} else {
		return os.OpenFile(_name, flag, perm)
	}
}

func (fs *FileSystem) RemoveAll(path string) error {
	fmt.Fprintf(fs.log, "Removing file: %v\n", path)
	if fs.readonly {
		return os.ErrPermission
	}
	_path := fmt.Sprintf("%v/%v", fs.root, path)
	return os.RemoveAll(_path)
}

func (fs *FileSystem) Rename(oldpath, newpath string) error {
	fmt.Fprintf(fs.log, "Renaming file: %v to %v\n", oldpath, newpath)
	if fs.readonly {
		return os.ErrPermission
	}
	_oldpath := fmt.Sprintf("%v/%v", fs.root, oldpath)
	_newpath := fmt.Sprintf("%v/%v", fs.root, newpath)
	return os.Rename(_oldpath, _newpath)
}

func (fs *FileSystem) Stat(name string) (os.FileInfo, error) {
	fmt.Fprintf(fs.log, "Stat on file: %v\n", name)
	_name := fmt.Sprintf("%v/%v", fs.root, name)
	return os.Stat(_name)
}

func (fs *FileSystem) ReadOnly(readonly *bool) bool {
	if readonly != nil {
		fs.readonly = *readonly
	}
	return fs.readonly
}
