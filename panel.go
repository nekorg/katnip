package katnip

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

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
	reader     io.Reader
}

type PanelHandler interface {
	Run(k *Kitty, w io.Writer)
}

type PanelFunc func(k *Kitty, w io.Writer)

func (f PanelFunc) Run(k *Kitty, w io.Writer) {
	f(k, w)
}

//go:generate stringer -type=Layer -linecomment
type Layer int

const (
	LayerBackground Layer = iota + 1 // background
	LayerBottom                      // bottom
	LayerTop                         // top
	LayerOverlay                     // overlay
)

//go:generate stringer -type=FocusPolicy -linecomment
type FocusPolicy int

const (
	FocusExclusive  FocusPolicy = iota + 1 // exclusive
	FocusNotAllowed                        // not-allowed
	FocusOnDemand                          // on-demand
)

//go:generate stringer -type=Edge -linecomment
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

	// WithSignals causes Panel to install signal handlers to cancel panel context.
	// Specifically os.Interrupt (which is SIGINT for most systems), and SIGHUP
	// on receving these signals, the panel process will be terminated and context
	// be cancelled.
	// This differs from calling Panel.Stop since Panel.Stop tries to gracefully shutdown
	// the child panel process first before
	WithSignals bool
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

	args = append(args, fmt.Sprintf("/proc/%d/exe", os.Getpid()))

	cmd := exec.Command(kittyCmd, args...)

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

	shmStream, err := shmstream.New()
	if err == nil {
		p.shmStream = shmStream
		cmd.Env = append(cmd.Env, GetEnvPair("SHM_PATH", shmStream.Path()))
		if reader, err := shmStream.NewReader(); err == nil {
			p.reader = reader
		}
	}
	return p
}

func (p *Panel) cleanup() {
	if p.shmStream != nil {
		p.shmStream.Close()
		p.shmStream = nil
		p.reader = nil
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
	return p.reader
}

// ReadOutput reads all available output from the panel
// Returns empty slice if no shared memory reader is available
func (p *Panel) ReadOutput() ([]byte, error) {
	if p.reader == nil {
		return []byte{}, nil
	}

	// Read available data
	buf := make([]byte, 4096)
	n, err := p.reader.Read(buf)
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
