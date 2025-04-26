package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
)

const (
	kittyCmd = "kittyn"
)

type KittyWindow struct {
	process    *exec.Cmd
	socketPath string
	x, y, w, h int

	done chan bool
}

func (w *KittyWindow) RemoteControl(args ...string) *exec.Cmd {
	base := []string{
		"@",
		"--to", w.socketPath,
	}
	c := exec.Command(kittyCmd, append(base, args...)...)
	return c
}

func (w *KittyWindow) Close() {
	w.RemoteControl("close-window").Run()
	w.done <- true
}

func createKittyWindow(x, y, w, h int) *KittyWindow {
	socketPath := fmt.Sprintf("unix:/tmp/katnip-%d", os.Getpid())
	args := []string{
		"+kitten", "panel",
		"--edge", "none",
		"--layer", "top",
		"--focus-policy", "on-demand",
		"--listen-on", socketPath,
		"-o", "allow_remote_control=socket-only",
		"--lines", strconv.Itoa(h),
		"--columns", strconv.Itoa(w),
		"--margin-top", strconv.Itoa(y),
		"--margin-left", strconv.Itoa(x),
		"box",
	}
	c := exec.Command(kittyCmd, args...)
	d := make(chan bool)
	go func() {
		c.Run()
		d <- true
	}()
	return &KittyWindow{c, socketPath, x, y, w, h, d}
}

func main() {
	w := createKittyWindow(300, 400, 30, 20)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGHUP)
	go func() {
		for range stop {
			w.Close()
			break
		}
		w.done <- true
	}()

	<-w.done
}
