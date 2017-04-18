# go-webdav-parallel
A golang webdav implementation that has a LRU  cache and prefetching with parallel file reads, primarily to speed up reading from mounted shares of distributed filesystems like cephfs.
