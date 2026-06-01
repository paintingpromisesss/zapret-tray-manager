package assets

import _ "embed"

//go:embed rkn_icon.ico
var IconStopped []byte

//go:embed rkn_blocked_icon.ico
var IconRunning []byte
