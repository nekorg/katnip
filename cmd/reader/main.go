package main

import (
	"fmt"
	"io"
	"os"

	"github.com/codelif/katnip/internal/shmlog"
	"golang.org/x/sys/unix"
)

func must(a any, b ...any) {
	err, ok := a.(error)
	if ok && err != nil {
		panic(err)
	}

	for _, e := range b {
		err, ok = e.(error)
		if ok && err != nil {
			panic(err)
		}
	}
}

func main() {
	l, err := shmlog.New()
	must(err)
	defer unix.Close(l.Fd)
	fmt.Printf("/proc/%d/fd/%d\n", os.Getpid(), l.Fd)

	r, err := l.NewReader()
	must(err)

	io.Copy(os.Stdout, r)
}
