package katnip

import (
	"io"
)

type Panel interface {
	Run(k *Kitty, w io.Writer)
}

type PanelFunc func(k *Kitty, w io.Writer)

func (f PanelFunc) Run(k *Kitty, w io.Writer) {
	f(k, w)
}

//go:generate stringer -type=Layer -linecomment
type Layer int

const (
	LayerBackground Layer = iota // background
	LayerBottom                  // bottom
	LayerTop                     // top
	LayerOverlay                 // overlay
)

//go:generate stringer -type=FocusPolicy -linecomment
type FocusPolicy int

const (
	FocusExclusive  FocusPolicy = iota // exclusive
	FocusNotAllowed                    // not-allowed
	FocusOnDemand                      // on-demand
)

//go:generate stringer -type=Edge -linecomment
type Edge int

const (
	EdgeBackground  Edge = iota // background
	EdgeBottom                  // bottom
	EdgeCenter                  // center
	EdgeCenterSized             // center-sized
	EdgeLeft                    // left
	EdgeNone                    // none
	EdgeRight                   // right
	EdgeTop                     // top
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
