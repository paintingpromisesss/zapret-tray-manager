//go:build !windows

package tray

func infoDialog(title, instruction, content string) {
	legacyInfo(title, instruction, content)
}

func errorDialog(title, instruction, content string) {
	legacyError(title, instruction, content)
}
