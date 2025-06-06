package pusuclt

import (
	"fmt"
	"os"
	"os/user"
)

const clientIDSep = ";"

const (
	clientIDProgramPfx = "program: "
	clientIDHostPfx    = "host: "
	clientIDUserPfx    = "user: "
	clientIDPidPfx     = "pid: "
)

// getClientIDPartHostname returns the hostname or a blank string if
// os.Hostname returns an error
func getClientIDPartHostname() string {
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}

	return ""
}

// getClientIDPartUser returns the user details or a blank string if
// user.Current returns an error
func getClientIDPartUser() string {
	if user, err := user.Current(); err == nil {
		return user.Uid + "/" +
			user.Gid + "/" +
			user.Username + "(" + user.Name + ")"
	}

	return ""
}

// makeClientID returns client ID details consisting of the supplied
// progname, the hostname, user details and the process ID. Note that none of
// this information is verified and so the pub/sub server should only use
// this for display not for security purposes.
func makeClientID(progName string) string {
	id := clientIDProgramPfx + progName
	id += clientIDSep
	id += clientIDHostPfx + getClientIDPartHostname()
	id += clientIDSep
	id += clientIDUserPfx + getClientIDPartUser()
	id += clientIDSep
	id += clientIDPidPfx + fmt.Sprintf("%d", os.Getpid())

	return id
}
