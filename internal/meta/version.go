package meta

import (
	"runtime"
)

const Version = "2.2.0"

const UserAgent = "CF-DDNS/" + Version + " (" + runtime.GOOS + "; " + runtime.GOARCH + ")"
