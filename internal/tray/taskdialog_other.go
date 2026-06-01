//go:build !windows

package tray

func confirmDialog(title, instruction, content string) bool {
	return legacyConfirm(title, instruction, content)
}

func infoDialog(title, instruction, content string) {
	legacyInfo(title, instruction, content)
}

func errorDialog(title, instruction, content string) {
	legacyError(title, instruction, content)
}
