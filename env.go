package katnip

import "fmt"

const envname = "KATNIP"

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
