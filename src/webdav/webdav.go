package main

import (
	"flag"
	"fmt"
	"golang.org/x/net/webdav"
	"log"
	"net/http"
	"os"
)

const APP_VERSION = "0.1"

// The flag package provides a default help printer via -h switch
var versionFlag *bool = flag.Bool("v", false, "Print the version number.")
var root *string = flag.String("r", "./", "The root of the webdav share")
var size *int = flag.Int("s", 128, "If larger than 0, enable caching filesystem and set the cache size (number of 4MB objects)")
var prefetch *int64 = flag.Int64("p", 64, "Prefetch this number of blocks")
var readOnly *bool = flag.Bool("ro", false, "Readonly filesystem")
var debug *bool = flag.Bool("d", false, "Debug logging")

func main() {
	flag.Parse() // Scan the arguments list

	if *versionFlag {
		fmt.Println("Version:", APP_VERSION)
		return
	}

	fs, err := NewFileSystem(*root, *size, *prefetch)
	if err != nil {
		panic(err)
	}
	fs.ReadOnly(readOnly)
	fs.SetLog(os.Stderr)
	if *debug {
		fs.SetDebugLog(os.Stderr)
	}

	srv := &webdav.Handler{
		Prefix:     "/",
		FileSystem: fs,
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			log.Printf("WEBDAV: %#s, ERROR: %v", r, err)
		},
	}
	http.Handle("/", srv)
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatalf("Error with WebDAV server: %v", err)
	}
}
