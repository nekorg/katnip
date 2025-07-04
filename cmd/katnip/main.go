package main

import (
	"fmt"
	"io"
	"time"

	"github.com/codelif/katnip"
)

func main() {
	katnip.RegisterFunc("clock", clock)

	katnip.Launch("clock", katnip.Config{Size: katnip.Vector{X: 0, Y: 3}, FocusPolicy: katnip.FocusOnDemand})
}

func clock(k *katnip.Kitty, w io.Writer) {
	for i := range 100 {
		fmt.Fprintf(w, "%s,i=%d               \r", time.Now().String(), i)
		time.Sleep(time.Second)
    if i == 5 {
      k.Dispatch("set-font-size", map[string]int{"size":20})
    }
	}
}
