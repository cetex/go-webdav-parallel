package main

import (
	"github.com/hashicorp/golang-lru"
	"sync"
)

type CacheObject struct {
	object *[]byte
	name   string
	sync.RWMutex
}

func NewCacheObject() *CacheObject {
	obj := new(CacheObject)
	return obj
}

type Cache struct {
	lru.Cache
	sync.RWMutex
	prefetch int64
}
