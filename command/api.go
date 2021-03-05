package command

import (
	"fmt"
)

// API command
type API struct {
	Command    string
	Arguments  string
	Background bool
}

// BuildMessage Implement API command interface
func (api API) BuildMessage() string {
	if api.Background {
		return fmt.Sprintf("bgapi %s %s", api.Command, api.Arguments)
	}
	return fmt.Sprintf("api %s %s", api.Command, api.Arguments)
}
