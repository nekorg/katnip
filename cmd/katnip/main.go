package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/codelif/katnip"
)

func init() {
	katnip.RegisterFunc("clock", clock)
	katnip.RegisterFunc("clock2", clock2)
	katnip.RegisterFunc("logger", loggerPanel)
}

func main() {
	// Example: Your clock2 case - clean and simple!
	// asyncExample()
  // advancedExample()
  // contextExample()
  multiPanelExample()
}

func clock(k *katnip.Kitty, w io.Writer) {
	for i := range 15 {
		fmt.Fprintf(os.Stdout, "%s,i=%d               \r", time.Now().String(), i)
		time.Sleep(time.Second)
		if i == 5 {
			k.Dispatch("set-font-size", map[string]int{"size": 20})
		}
	}
}

// Your clock2 function - now clean and intuitive!
func clock2(k *katnip.Kitty, w io.Writer) {
	// Create a panel
	clockPanel := katnip.NewPanel("clock", katnip.Config{
		Size:        katnip.Vector{X: 0, Y: 5},
		FocusPolicy: katnip.FocusOnDemand,
	})
	
	// Start it asynchronously
	if err := clockPanel.Start(); err != nil {
		fmt.Fprintf(w, "Failed to start clock: %v\n", err)
		return
	}
	
	// Do our work
	for i := range 10 {
		fmt.Fprintf(os.Stdout, "clock2: %s,i=%d               \r", time.Now().String(), i)
		time.Sleep(time.Second)
		if i == 5 {
			k.Dispatch("set-font-size", map[string]int{"size": 20})
		}
	}
	
	// Wait for the clock panel to finish
	clockPanel.Wait()
}

// Example 1: Simple blocking execution
func simpleExample() {
	panel := katnip.TopPanel("clock", 3)
	panel.Run() // Blocks until panel exits
}

// Example 2: Async execution
func asyncExample() {
	// Create panel
	panel := katnip.NewPanel("clock2", katnip.Config{
		Size:        katnip.Vector{Y: 3},
		FocusPolicy: katnip.FocusOnDemand,
	})
	
	// Start async
	if err := panel.Start(); err != nil {
		panic(err)
	}
	
	fmt.Println("Panel is running in background")
	
	// Do other work...
	time.Sleep(5 * time.Second)
	
	// Wait for completion
	panel.Wait()
}

// Example 3: Advanced - direct Cmd access
func advancedExample() {
	panel := katnip.BottomPanel("logger", 10)
	
	// Advanced users can access Cmd directly
	panel.Cmd.Env = append(panel.Cmd.Env, "DEBUG=true")
	panel.Cmd.Dir = "/tmp"
	
	// Set up pipes if needed
	panel.Cmd.Stderr = os.Stderr
	
	// Start the panel
	if err := panel.Start(); err != nil {
		panic(err)
	}
	
	// Let it run
	time.Sleep(10 * time.Second)
	
	// Graceful shutdown
	panel.Stop()
	panel.Wait()
}

// Example 4: Multiple panels
func multiPanelExample() {
	// Create multiple panels
	statusBar := katnip.TopPanel("clock", 2)
	sidePanel := katnip.FloatingPanel("logger", 20, 40)
	
	// Start them
	statusBar.Start()
	sidePanel.Start()
	
	fmt.Println("Both panels running. Press Enter to stop...")
	fmt.Scanln()
	
	// Stop them gracefully
	statusBar.Stop()
	sidePanel.Stop()
	
	// Wait for cleanup
	statusBar.Wait()
	sidePanel.Wait()
}

// Example 5: Context with timeout
func contextExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Create context-aware panel
	panel := katnip.NewPanelContext(ctx, "clock", katnip.Config{
		Size: katnip.Vector{Y: 3},
	})
	
	// Will automatically be killed when context expires
	panel.Run()
}

func loggerPanel(k *katnip.Kitty, w io.Writer) {
	for i := 0; i < 20; i++ {
		fmt.Fprintf(os.Stdout, "Log message %d\n", i)
		time.Sleep(500 * time.Millisecond)
	}
}
