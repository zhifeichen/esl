package command

import "fmt"

// Auth user auth info
type Auth struct {
	User string
	Passwd string
}

// BuildMessage Implement command interface
func (auth Auth) BuildMessage() string {
	if len(auth.User) > 0 {
		return fmt.Sprintf("userauth %s:%s", auth.User, auth.Passwd)
	}
	return fmt.Sprintf("auth %s", auth.Passwd)
}
