package command

// Exit command
type Exit struct{}

// BuildMessage Implement command interface
func (Exit) BuildMessage() string {
	return "exit"
}
