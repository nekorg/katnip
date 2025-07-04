package katnip

import (
	"errors"
	"os/exec"
	"strings"
	"time"
)


type NotificationWriter struct{}

func (w *NotificationWriter) Write(p []byte) (n int, err error) {
	cmd := exec.Command("notify-send", strings.TrimRight(string(p), "\n"))

	done := make(chan error)
	go func() {
		err := cmd.Run()
		done <- err
	}()

	select {
	case <-time.After(1 * time.Second):
		return 0, errors.New("Command timed out.")
	case d := <-done:
		if d != nil {
			return 0, d
		}
		return len(p), d
	}
}
