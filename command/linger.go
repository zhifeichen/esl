package command

import "fmt"

// Linger command
type Linger struct{
	Enabled bool
	Duration int
}

// BuildMessage Implement command interface
func (l Linger) BuildMessage() string {
	if l.Enabled {
		if l.Duration > 0 {
			return fmt.Sprintf("linger %d", l.Duration)
		}
		return "linger"
	}
	return "nolinger"
}
