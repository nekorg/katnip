// Copyright (c) 2025 Harsh Sharma <harsh@codelif.in>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package katnip

import (
	"fmt"
	"io"
	"os"

	"github.com/codelif/shmstream"
)

var registry = map[string]PanelHandler{}

func Register(name string, panel PanelHandler) {
	instance := os.Getenv(GetEnvKey("INSTANCE"))
	if instance != "" && instance == name {
		panelExitCode, err := runPanel(panel)
		if err != nil {
			os.Exit(1)
		}
		os.Exit(panelExitCode)
	}

	registry[name] = panel
}

func RegisterFunc(name string, panel PanelFunc) {
	Register(name, panel)
}

func runPanel(panel PanelHandler) (int, error) {
	socketPath := os.Getenv(GetEnvKey("SOCKET"))
	if socketPath == "" {
		return -1, fmt.Errorf("Kitty socket path not given")
	}

	shmPath := os.Getenv(GetEnvKey("SHM_PATH"))
	var shmIo io.ReadWriter
	if shmPath != "" {
		shmBuf, err := shmstream.Open(shmPath)
		if err != nil {
			return -1, fmt.Errorf("failed to open shared memory: %w", err)
		}

		defer shmBuf.Close()

		writer, err := shmBuf.NewWriter()
		if err != nil {
			return -1, fmt.Errorf("failed to create shared memory writer: %w", err)
		}
		reader, err := shmBuf.NewReader()
		if err != nil {
			return -1, fmt.Errorf("failed to create shared memory reader: %w", err)
		}
		shmIo = &struct {
			io.Reader
			io.Writer
		}{reader, writer}
	}
	k := NewKitty(socketPath)

	return panel.Run(k, shmIo), nil
}

// Convenience constructors for common panel types

// TopPanel creates a top-edge panel
func TopPanel(name string, lines int) *Panel {
	return NewPanel(name, Config{
		Edge: EdgeTop,
		Size: Vector{Y: lines},
	})
}

// BottomPanel creates a bottom-edge panel
func BottomPanel(name string, lines int) *Panel {
	return NewPanel(name, Config{
		Edge: EdgeBottom,
		Size: Vector{Y: lines},
	})
}

// BackgroundPanel creates a background/wallpaper panel
func BackgroundPanel(name string) *Panel {
	return NewPanel(name, Config{
		Edge:  EdgeBackground,
		Layer: LayerBackground,
	})
}

// FloatingPanel creates a centered floating panel
func FloatingPanel(name string, lines, columns int) *Panel {
	return NewPanel(name, Config{
		Edge: EdgeCenter,
		Size: Vector{X: columns, Y: lines},
	})
}

func Launch(name string, config Config) {
	NewPanel(name, config).Run()
}
