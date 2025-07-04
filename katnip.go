package katnip

import (
	"fmt"
	"io"
	"os"

)

var registry = map[string]PanelHandler{}

func Register(name string, panel PanelHandler) {
	instance := os.Getenv(GetEnvKey("INSTANCE"))
	if instance != "" && instance == name {
		if err := runPanel(panel); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	registry[name] = panel
}

func RegisterFunc(name string, panel PanelFunc) {
	Register(name, panel)
}

func runPanel(panel PanelHandler) error {
	socketPath := os.Getenv(GetEnvKey("SOCKET"))
	if socketPath == "" {
		return fmt.Errorf("Kitty socket path not given")
	}

	k := &Kitty{socketPath}
	w := &NotificationWriter{}

	panel.Run(k, w)

	return nil
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
