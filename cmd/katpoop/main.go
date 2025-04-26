package main

import (
	"fmt"

	"git.sr.ht/~rockorager/vaxis"
)

func main() {
	vx, err := vaxis.New(vaxis.Options{EnableSGRPixels: true})
	if err != nil {
		panic(err)
	}
	defer vx.Close()

	win := vx.Window()
	win.Clear()
	win.Print(vaxis.Segment{Text: "Move the mouse to start..."})
	for ev := range vx.Events() {
		switch ev := ev.(type) {
		case vaxis.Key:
			switch ev.String() {
			case "Ctrl+c":
				return
			}
		case vaxis.Mouse:
			win.Clear()
			win.Print(vaxis.Segment{Text: fmt.Sprintf("%d, %d\n%d, %d", ev.Col, ev.Row, ev.XPixel, ev.YPixel)})
		}
		vx.Render()
	}
}
