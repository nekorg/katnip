package main

import (
	"fmt"
	"io"
	"time"

	"github.com/codelif/katnip"
)

func init() {
	katnip.RegisterFunc("logger", loggerPanel)
	katnip.RegisterFunc("writer", writerPanel)
}

func main() {
	// Example showing how to read panel output
	readPanelOutput()
}

// Panel that writes to the shared memory stream
func writerPanel(k *katnip.Kitty, w io.Writer) {
	// w is now a shared memory writer!
	for i := 0; i < 10; i++ {
		msg := fmt.Sprintf("Message %d from panel at %s\n", i, time.Now().Format("15:04:05"))
		
		// Write to shared memory
		if _, err := w.Write([]byte(msg)); err != nil {
			fmt.Printf("Error writing to shared memory: %v\n", err)
		}
		
		// Also print to stdout for visual feedback
		fmt.Print(msg)
		
		time.Sleep(1 * time.Second)
	}
}

// Example of reading panel output from parent process
func readPanelOutput() {
	// Create panel
	panel := katnip.NewPanel("writer", katnip.Config{
		Size: katnip.Vector{Y: 5},
	})
	
	// Start panel
	if err := panel.Start(); err != nil {
		panic(err)
	}
	
	// Read output from panel in a goroutine
	go func() {
		reader := panel.Reader()
		if reader == nil {
			fmt.Println("No shared memory reader available")
			return
		}
		
		buf := make([]byte, 1024)
		for {
			n, err := reader.Read(buf)
			if err != nil && err != io.EOF {
				fmt.Printf("Error reading: %v\n", err)
				break
			}
			
			if n > 0 {
				fmt.Printf("Parent received: %s", string(buf[:n]))
			}
			
			time.Sleep(100 * time.Millisecond)
		}
	}()
	
	// Wait for panel to finish
	panel.Wait()
}

// Example with multiple panels communicating
func multiPanelCommunication() {
	// Writer panel
	writer := katnip.NewPanel("writer", katnip.Config{
		Size: katnip.Vector{Y: 5},
		Edge: katnip.EdgeTop,
	})
	
	// Logger panel
	logger := katnip.NewPanel("logger", katnip.Config{
		Size: katnip.Vector{Y: 5},
		Edge: katnip.EdgeBottom,
	})
	
	// Start both panels
	writer.Start()
	logger.Start()
	
	// Monitor writer output
	go func() {
		if reader := writer.Reader(); reader != nil {
			buf := make([]byte, 1024)
			for {
				n, err := reader.Read(buf)
				if err == nil && n > 0 {
					fmt.Printf("[Monitor] Writer said: %s", string(buf[:n]))
				}
			}
		}
	}()
	
	// Wait for both
	writer.Wait()
	logger.Wait()
}

func loggerPanel(k *katnip.Kitty, w io.Writer) {
	// This panel also gets a shared memory writer
	w.Write([]byte("Logger panel started\n"))
	
	for i := 0; i < 5; i++ {
		msg := fmt.Sprintf("Log entry %d\n", i)
		w.Write([]byte(msg))
		fmt.Print(msg)
		time.Sleep(2 * time.Second)
	}
	
	w.Write([]byte("Logger panel finished\n"))
}
