package katnip

import (
	"encoding/json"
	"fmt"
	"net"
)

type Kitty struct {
	socketPath string
}

type kittySockMsg struct {
	Command       string          `json:"cmd"`
	Version       [3]uint64       `json:"version"`
	NoResponse    bool            `json:"no_response,omitempty"`
	KittyWindowId uint64          `json:"kitty_window_id,omitempty"`
	Payload       json.RawMessage `json:"payload,omitempty"`
}

var kittyRCHeader = append([]byte{0x1b}, "P@kitty-cmd"...)

func packMsg(msg []byte) []byte {
	return append(append(kittyRCHeader, msg...), 0x1b, '\\')
}

// For dispatching commands only, no response is assumed. For $(kitty --version) > v0.42.0
func (k *Kitty) Dispatch(cmd string, payload any) error {
	// new dispatch, new socket; coz I am too lazy,
	// and its a pain to manage. Though I'll keep a todo below:
	// TODO: keep only one socket connection for Panel's lifetime
	sock, err := net.Dial("unix", k.socketPath)
	if err != nil {
		return err
	}
	defer sock.Close()

	var p []byte
	// easiest way to induce omitempty for payload
	if payload != nil {
		p, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}

	mraw := kittySockMsg{Command: cmd, Version: [3]uint64{0, 42, 0}, Payload: p}

	msg, err := json.Marshal(mraw)
	if err != nil {
		return err
	}

	_, err = sock.Write(packMsg(msg))
	if err != nil {
		return err
	}

	resp_header := make([]byte, len(kittyRCHeader))
	sock.Read(resp_header)

	var resp map[string]any
	dec := json.NewDecoder(sock)
	err = dec.Decode(&resp)
	if err != nil {
		return err
	}

	resp_ok_any, ok := resp["ok"]
	if !ok {
		return fmt.Errorf("invalid response schema: field 'ok' not found")
	}

	resp_ok, ok := resp_ok_any.(bool)
	if !ok {
		return fmt.Errorf("invalid response schema: field 'ok' not a boolean")
	}

	if !resp_ok {
		resp_error_any, ok := resp["error"]
		if !ok {
			return fmt.Errorf("invalid response schema: field 'error' not found")
		}

		resp_error, ok := resp_error_any.(string)
		if !ok {
			return fmt.Errorf("invalid response schema: field 'error' not a string")
		}

		return fmt.Errorf(resp_error)
	}

	return nil
}
