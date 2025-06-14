package main

import (
	"fmt"
	"io"
	"os"

	"github.com/codelif/katnip/internal/shmlog"
	"golang.org/x/sys/unix"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s /proc/<pid>/fd/<fd>\n", os.Args[0])
		os.Exit(1)
	}
	path := os.Args[1]

	fd, err := unix.Open(path, unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open %s: %v\n", path, err)
		os.Exit(1)
	}
	defer unix.Close(fd)

	fmt.Printf("opened FD %d (path %s)\n", fd, path)

	l := shmlog.Logger{Fd: fd}
	w, err := l.NewWriter()
	if err != nil {
		panic(err)
	}

  io.Copy(w, os.Stdin)
}
