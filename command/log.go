package command

import "fmt"

// Log command
type Log struct{
	Enabled bool
	Level int
}

// BuildMessage Implement command interface
func (l Log) BuildMessage() string {
	if l.Enabled {
		return fmt.Sprintf("log %d", l.Level)
	}
	return "nolog"
}
