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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
)

const esc byte = 0x1b

var (
	kittyMsgPrefix  = []byte("\x1bP@kitty-cmd")
	kittyMsgSuffix  = []byte("\x1b\\")
	kittyMinVersion = [3]uint64{0, 42, 0}
)

type kittySockMsg struct {
	Command       string          `json:"cmd"`
	Version       [3]uint64       `json:"version"`
	NoResponse    bool            `json:"no_response,omitempty"`
	KittyWindowId uint64          `json:"kitty_window_id,omitempty"`
	Payload       json.RawMessage `json:"payload,omitempty"`
}

type Kitty struct {
	socketPath string
	conn       net.Conn
	reader     *bufio.Reader
	mu         sync.Mutex
	connected  bool
}

func NewKitty(socketPath string) *Kitty {
	return &Kitty{socketPath: socketPath}
}

func (k *Kitty) connect() error {
	if k.connected {
		return nil
	}
	conn, err := net.Dial("unix", k.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to kitty socket: %w", err)
	}

	k.conn = conn
	k.reader = bufio.NewReader(conn)
	k.connected = true

	return nil
}

func (k *Kitty) ensureConnected() error {
	if k.connected && k.conn != nil {
		return nil
	}

	return k.connect()
}

func (k *Kitty) Close() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if !k.connected || k.conn == nil {
		return nil
	}

	err := k.conn.Close()
	k.conn = nil
	k.reader = nil
	k.connected = false

	return err
}

func packMsg(msg []byte) []byte {
	return append(append(kittyMsgPrefix, msg...), kittyMsgSuffix...)
}

func (k *Kitty) readFrame() ([]byte, error) {
	var buf bytes.Buffer
	for {
		b, err := k.reader.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("reading frame: %w", err)
		}

		if b == esc {
			next, err := k.reader.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("reading frame: %w", err)
			}
			if next == '\\' {
				return buf.Bytes(), nil
			}

			buf.WriteByte(b)
			buf.WriteByte(next)
			continue
		}
		buf.WriteByte(b)
	}
}

// For dispatching commands only, no response is assumed. For $(kitty --version) > v0.42.0
func (k *Kitty) Dispatch(cmd string, payload any) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if err := k.ensureConnected(); err != nil {
		return err
	}

	_, err := k.command(cmd, payload)
	return err
}

// Like Dispatch but response is returned.
func (k *Kitty) Command(cmd string, payload any) (map[string]any, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	if err := k.ensureConnected(); err != nil {
		return nil, err
	}

	return k.command(cmd, payload)
}

func (k *Kitty) command(cmd string, payload any) (map[string]any, error) {
	var p []byte
	var err error
	// easiest way to induce omitempty for payload
	if payload != nil {
		p, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to mashal payload: %w", err)
		}
	}

	msg := kittySockMsg{
		Command: cmd,
		Version: kittyMinVersion,
		Payload: p,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to mashal message: %w", err)
	}

	_, err = k.conn.Write(packMsg(msgBytes))
	if err != nil {
		k.connected = false
		return nil, fmt.Errorf("failed to write message: %w", err)
	}

	if _, err = io.ReadFull(k.reader, make([]byte, len(kittyMsgPrefix))); err != nil {
		k.connected = false
		return nil, fmt.Errorf("failed to read response header: %w", err)
	}

	respBytes, err := k.readFrame()
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var resp map[string]any
	if err = json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	respOk, ok := resp["ok"].(bool)
	if !ok {
		return nil, fmt.Errorf("invalid response schema: field 'ok' is missing/invalid.")
	}

	if !respOk {
		respError, ok := resp["error"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid response schema: field 'error' is missing/invalid")
		}

		return nil, fmt.Errorf("kitty error: %s", respError)
	}

	return resp, nil
}

func (k *Kitty) SetFontSize(size int) error {
	return k.Dispatch("set-font-size", map[string]int{"size": size})
}

func (k *Kitty) SetOpacity(opacity float64) error {
	return k.Dispatch("set-background-opacity", map[string]float64{"opacity": opacity})
}

func (k *Kitty) Resize(columns, lines int) error {
	return k.Dispatch("resize-os-window", map[string]any{
		"action": "os-panel",
    "incremental": true,
		"os_panel": []string{
			fmt.Sprintf("lines=%d", lines),
			fmt.Sprintf("columns=%d", columns),
			// fmt.Sprintf("edge=%s", edge),
			// fmt.Sprintf("layer=%s", layer),
		},
	})
}

func (k *Kitty) Move(x, y int) error {
	return k.Dispatch("resize-os-window", map[string]any{
		"action": "os-panel",
    "incremental": true,
		"os_panel": []string{
			fmt.Sprintf("margin-left=%d", x),
			fmt.Sprintf("margin-top=%d", y),
			// fmt.Sprintf("edge=%s", edge),
			// fmt.Sprintf("layer=%s", layer),
		},
	})
}

func (k *Kitty) Show() error {
	return k.Dispatch("resize-os-window", map[string]string{
		"action": "show",
	})
}

func (k *Kitty) Hide() error {
	return k.Dispatch("resize-os-window", map[string]string{
		"action": "hide",
	})
}

func (k *Kitty) ToggleVisibility() error {
	return k.Dispatch("resize-os-window", map[string]string{
		"action": "toggle-visibility",
	})
}
