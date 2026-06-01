package tray

import (
	"strings"

	"github.com/sqweek/dialog"
)

func legacyBody(instruction, content string) string {
	switch {
	case instruction == "":
		return content
	case content == "":
		return instruction
	default:
		return instruction + "\n\n" + content
	}
}

func legacyInfo(title, instruction, content string) {
	dialog.Message("%s", legacyBody(instruction, content)).Title(title).Info()
}

func legacyError(title, instruction, content string) {
	dialog.Message("%s", strings.TrimSpace(legacyBody(instruction, content))).Title(title).Error()
}
