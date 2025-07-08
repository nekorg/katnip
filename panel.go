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

//go:generate stringer -type=Layer,FocusPolicy,Edge -linecomment -output panel_string.go
package katnip

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/codelif/shmstream"
)

type Panel struct {
	// pointer to underlying exec.Cmd, can be used to set Env
	// or directly interface with exec.Cmd
	Cmd *exec.Cmd

	name       string
	config     Config
	socketPath string
	started    bool
	shmStream  *shmstream.StreamBuffer
	shmIo      io.ReadWriter
}

type PanelHandler interface {
	Run(k *Kitty, rw io.ReadWriter) int
}

type PanelFunc func(k *Kitty, rw io.ReadWriter) int

func (f PanelFunc) Run(k *Kitty, rw io.ReadWriter) int {
	return f(k, rw)
}

type Layer int

const (
	LayerBackground Layer = iota + 1 // background
	LayerBottom                      // bottom
	LayerTop                         // top
	LayerOverlay                     // overlay
)

type FocusPolicy int

const (
	FocusExclusive  FocusPolicy = iota + 1 // exclusive
	FocusNotAllowed                        // not-allowed
	FocusOnDemand                          // on-demand
)

type Edge int

const (
	EdgeBackground  Edge = iota + 1 // background
	EdgeBottom                      // bottom
	EdgeCenter                      // center
	EdgeCenterSized                 // center-sized
	EdgeLeft                        // left
	EdgeNone                        // none
	EdgeRight                       // right
	EdgeTop                         // top
)

type Vector struct {
	X, Y int
}
type Config struct {
	Position    Vector
	Size        Vector
	Layer       Layer
	FocusPolicy FocusPolicy
	Edge        Edge
	ConfigFile  string
	Overrides   []string
}

const kittyCmd = "kitty"

var index uint64 = 0

func NewPanel(name string, config Config) *Panel {
	socketPath := fmt.Sprintf("/tmp/katnip-%s-%d-%d", name, os.Getpid(), index)
	index++

	args := []string{
		"+kitten", "panel",
		"--listen-on", "unix:" + socketPath,
		"-o", "allow_remote_control=socket-only",
	}

	if config.Layer > 0 {
		args = append(args, "--layer", config.Layer.String())
	}
	if config.FocusPolicy > 0 {
		args = append(args, "--focus-policy", config.FocusPolicy.String())
	}
	if config.Edge > 0 {
		args = append(args, "--edge", config.Edge.String())
	}
	if config.Size.X > 0 {
		args = append(args, "--columns", strconv.Itoa(config.Size.X))
	}
	if config.Size.Y > 0 {
		args = append(args, "--lines", strconv.Itoa(config.Size.Y))
	}
	if config.Position.X > 0 {
		args = append(args, "--margin-left", strconv.Itoa(config.Position.X))
	}
	if config.Position.Y > 0 {
		args = append(args, "--margin-top", strconv.Itoa(config.Position.Y))
	}
	if config.ConfigFile != "" {
		args = append(args, "--config", config.ConfigFile)
	}

	for _, o := range config.Overrides {
		args = append(args, "-o", o)
	}

	args = append(args, fmt.Sprintf("/proc/%d/exe", os.Getpid()))

	cmd := exec.Command(kittyCmd, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}

	cmd.Env = append(os.Environ(),
		GetEnvPair("INSTANCE", name),
		GetEnvPair("SOCKET", socketPath),
	)

	p := &Panel{
		Cmd:        cmd,
		name:       name,
		config:     config,
		socketPath: socketPath,
	}

	shmStream, err := shmstream.New(shmstream.Config{Bidirectional: true})
	if err == nil {
		p.shmStream = shmStream
		cmd.Env = append(cmd.Env, GetEnvPair("SHM_PATH", shmStream.Path()))
		reader, _ := shmStream.NewReader()
		writer, _ := shmStream.NewWriter()
		p.shmIo = &struct {
			io.Reader
			io.Writer
		}{reader, writer}
	}
	return p
}

func (p *Panel) cleanup() {
	if p.shmStream != nil {
		p.shmStream.Close()
		p.shmStream = nil
		p.shmIo = nil
	}
}

func (p *Panel) Run() error {
	if p.started {
		return fmt.Errorf("panel already started")
	}
	p.started = true
	return p.Cmd.Run()
}

func (p *Panel) Start() error {
	if p.started {
		return fmt.Errorf("panel already started")
	}
	p.started = true

	return p.Cmd.Start()
}

func (p *Panel) Wait() error {
	return p.Cmd.Wait()
}

func (p *Panel) Stop() error {
	if p.Cmd.Process == nil {
		return fmt.Errorf("panel not started")
	}
	return p.Cmd.Process.Signal(os.Interrupt)
}

func (p *Panel) Kill() error {
	if p.Cmd.Process == nil {
		return fmt.Errorf("panel not started")
	}
	return p.Cmd.Process.Kill()
}

func (p *Panel) Reader() io.Reader {
	return p.shmIo
}

func (p *Panel) Writer() io.Writer {
	return p.shmIo
}

func (p *Panel) ReadWriter() io.ReadWriter {
	return p.shmIo
}

// ReadOutput reads all available output from the panel
// Returns empty slice if no shared memory reader is available
func (p *Panel) ReadOutput() ([]byte, error) {
	if p.shmIo == nil {
		return []byte{}, nil
	}

	// Read available data
	buf := make([]byte, 4096)
	n, err := p.shmIo.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return buf[:n], nil
}

func NewPanelContext(ctx context.Context, name string, config Config) *Panel {
	p := NewPanel(name, config)

	cmd := exec.CommandContext(ctx, p.Cmd.Args[0], p.Cmd.Args[1:]...)
	cmd.Env = p.Cmd.Env
	p.Cmd = cmd

	return p
}
