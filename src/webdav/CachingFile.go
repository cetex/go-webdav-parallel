package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"sync"
	"syscall"
)

const BLOCKSIZE = 1024 * 1024 * 4

type CachingFile struct {
	name         string
	flag         int
	perm         os.FileMode
	fs           *FileSystem
	pos          int64
	OrigFileInfo *os.FileInfo
	file         *os.File
	inode        uint64
	sync.Mutex
}

func NewCachingFile(name string, flag int, perm os.FileMode, fs *FileSystem) (*CachingFile, error) {
	file, err := os.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	inode, err := GetInode(&stat)
	if err != nil {
		return nil, err
	}
	cf := CachingFile{
		name:         name,
		flag:         flag,
		perm:         perm,
		file:         file,
		inode:        inode,
		fs:           fs,
		OrigFileInfo: &stat}
	return &cf, err

}

func GetInode(stat *os.FileInfo) (uint64, error) {
	stat_t, ok := (*stat).Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("Failed to stat file, Not a syscall.Stat_t")
	}
	return stat_t.Ino, nil
}

func GetBlockKey(f *CachingFile, block int64) string {
	return fmt.Sprintf("%v.%v", f.inode, block)
}

func (f *CachingFile) Close() error {
	return f.file.Close()
}

func (f *CachingFile) readObj(startpos int64, obj *CacheObject) {
	read := 0
	if f.OrigFileInfo == nil { // Sanity check to see if file's stat is set.
		panic("ORIGFILEINFO IS NIL!")
	}
	length := (*f.OrigFileInfo).Size()
	file, err := os.OpenFile(f.name, f.flag, f.perm)
	if err != nil {
		panic("Failed to open flie")
	}
	defer file.Close()

	// Check if reopened file has the same inode as original opened file.
	// fallback to sequential reading on original file handle if inode has changed or stat fails.
	stat, err := file.Stat()
	if err != nil {
		// Stat failed, fallback to originally opened filehandle instead.
		f.Lock()
		file = f.file
		defer f.Unlock()
	}
	inode, err := GetInode(&stat)
	if err != nil {
		panic(err)
	}
	if inode != f.inode {
		// Inode has changed, fallback to sequential reading through original filehandle instead
		f.Lock()
		file = f.file
		defer f.Unlock()
	}

	data := make([]byte, int64(math.Min(BLOCKSIZE, float64(length-startpos))))
	pos, err := file.Seek(startpos, 0)
	if err != nil {
		panic("Failed to seek to pos")
	}
	if pos != startpos {
		panic("didn't seek to right position")
	}

	for read < len(data) {
		fmt.Fprintf(f.fs.debuglog, "readObj: file length: %v, read: %v\n", length, read)
		tmpdata := make([]byte, len(data)-read)
		n, err := file.Read(tmpdata)
		if err != nil {
			panic(err)
		}
		copy(data[read:], tmpdata[:n])
		read += n
	}

	obj.object = &data
	obj.Unlock()
}

func (f *CachingFile) Read(p []byte) (n int, err error) {
	blocknr := int64(math.Floor(float64(f.pos) / float64(BLOCKSIZE)))
	blockpos := f.pos - (blocknr * BLOCKSIZE)
	fmt.Fprintf(f.fs.debuglog, "Reading block nr: %v, Pos: %v, blockpos: %v\n", blocknr, f.pos, blockpos)
	lastBlock := math.Floor(float64((*f.OrigFileInfo).Size() / BLOCKSIZE))
	for block := blocknr; block <= int64(math.Min(float64(blocknr+f.fs.cache.prefetch), float64(lastBlock))); block++ {
		blockKey := GetBlockKey(f, block)
		if !f.fs.cache.Contains(blockKey) {
			f.fs.cache.Lock()
			if !f.fs.cache.Contains(blockKey) {
				fmt.Fprintf(f.fs.debuglog, "Adding cache key: %v\n", blockKey)
				obj := NewCacheObject()
				obj.name = f.name
				obj.Lock()
				f.fs.cache.Add(blockKey, obj)
				go f.readObj(blocknr*BLOCKSIZE, obj)
				f.fs.cache.Unlock()
			}
		}
	}
	blockKey := GetBlockKey(f, blocknr)
	if obj, ok := f.fs.cache.Get(blockKey); ok {
		fmt.Fprintf(f.fs.debuglog, "Getting cache key: %v, pos: %v, len(p): %v\n", blockKey, f.pos, len(p))
		_obj, _ := obj.(*CacheObject)
		if !ok {
			fmt.Printf("%+v\n", _obj)
			return 0, fmt.Errorf("Failed to cast cache object")
		}
		_obj.RLock()
		defer _obj.RUnlock()
		n := copy(p, (*_obj.object)[blockpos:])
		if n == 0 {
			return 0, io.EOF
		}
		fmt.Fprintf(f.fs.debuglog, "Got: %v bytes, total: %v\n", n, f.pos+int64(n))
		f.pos = f.pos + int64(n)
		fmt.Fprintf(f.fs.debuglog, "F.pos after add: %v\n", f.pos)
		return n, nil

	}
	panic("Weird shit happens, we should never ever end up here")
	return n, fmt.Errorf("Weird shit happens")
}

func (f *CachingFile) Seek(offset int64, whence int) (int64, error) {
	pos, err := f.file.Seek(offset, whence)
	f.pos = pos
	return f.pos, err
}

func (f *CachingFile) Write(p []byte) (n int, err error) {
	return f.file.Write(p)
}

func (f *CachingFile) Readdir(count int) ([]os.FileInfo, error) {
	return f.file.Readdir(count)
}

func (f *CachingFile) Stat() (os.FileInfo, error) {
	return f.file.Stat()
}
