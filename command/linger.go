package command

// Linger command
type Linger struct{
	Enabled bool
}

// BuildMessage Implement command interface
func (l Linger) BuildMessage() string {
	if l.Enabled {
		return "linger"
	}
	return "nolinger"
}
