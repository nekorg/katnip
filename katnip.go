package katnip

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

var (
	envname         = "KATNIP"
	kittyCmd        = "kitty"
	registry        = map[string]Panel{}
	index    uint64 = 0
)

// Returns "{envname}_{name}".
//
// envname is KATNIP by default
func GetEnvKey(name string) string {
	return fmt.Sprintf("%s_%s", envname, name)
}

// Returns "{key}={value}".
//
// key is the returned value from GetEnvKey function
// with name as argument
func GetEnvPair(name, value string) string {
	return fmt.Sprintf("%s=%s", GetEnvKey(name), value)
}

func Register(name string, panel Panel) {
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


func runPanel(panel Panel) error {
	socketPath := os.Getenv(GetEnvKey("SOCKET"))
	if socketPath == "" {
		return fmt.Errorf("Kitty socket path not given")
	}

	k := &Kitty{socketPath}
	w := &NotificationWriter{}

	panel.Run(k, w)

	return nil
}

func Launch(name string, config Config) {
	socketPath := fmt.Sprintf("/tmp/katnip-%d-%s-%d", name, os.Getpid(), index)
	index++
	args := []string{
		"+kitten", "panel",
		"--focus-policy", config.FocusPolicy.String(),
		"--listen-on", "unix:" + socketPath,
		"-o", "allow_remote_control=socket-only",
		"--lines", strconv.Itoa(config.Size.Y),
		fmt.Sprintf("/proc/%d/exe", os.Getpid()),
	}

	c := exec.Command(kittyCmd, args...)
	c.Env = append(c.Environ(), GetEnvPair("INSTANCE", name), GetEnvPair("SOCKET", socketPath))
  c.Run()
}
