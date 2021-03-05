package command

// Connect command
type Connect struct{}

// BuildMessage Implement command interface
func (Connect) BuildMessage() string {
	return "connect"
}
