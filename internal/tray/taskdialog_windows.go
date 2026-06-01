//go:build windows

package tray

import (
	"syscall"
	"unsafe"
)

var (
	comctl32               = syscall.NewLazyDLL("comctl32.dll")
	procTaskDialogIndirect = comctl32.NewProc("TaskDialogIndirect")
)

const (
	tdcbfOKButton = 0x0004 // TDCBF_OK_BUTTON

	tdfAllowDialogCancellation  = 0x0008 // TDF_ALLOW_DIALOG_CANCELLATION
	tdfPositionRelativeToWindow = 0x1000 // TDF_POSITION_RELATIVE_TO_WINDOW
)

const (
	tdErrorIcon       = 0xFFFE // TD_ERROR_ICON
	tdInformationIcon = 0xFFFD // TD_INFORMATION_ICON
)

type taskDialogConfig struct {
	cbSize                  uint32
	hwndParent              uintptr
	hInstance               uintptr
	dwFlags                 uint32
	dwCommonButtons         uint32
	pszWindowTitle          *uint16
	pszMainIcon             uintptr
	pszMainInstruction      *uint16
	pszContent              *uint16
	cButtons                uint32
	pButtons                uintptr
	nDefaultButton          int32
	cRadioButtons           uint32
	pRadioButtons           uintptr
	nDefaultRadioButton     int32
	pszVerificationText     *uint16
	pszExpandedInformation  *uint16
	pszExpandedControlText  *uint16
	pszCollapsedControlText *uint16
	pszFooterIcon           uintptr
	pszFooter               *uint16
	pfCallback              uintptr
	lpCallbackData          uintptr
	cxWidth                 uint32
}

func taskDialog(title, instruction, content string, icon uintptr, buttons uint32) bool {
	if procTaskDialogIndirect.Find() != nil {
		return false
	}
	cfg := taskDialogConfig{
		dwFlags:            tdfAllowDialogCancellation | tdfPositionRelativeToWindow,
		dwCommonButtons:    buttons,
		pszWindowTitle:     mustUTF16(title),
		pszMainIcon:        icon,
		pszMainInstruction: mustUTF16(instruction),
		pszContent:         mustUTF16(content),
	}
	cfg.cbSize = uint32(unsafe.Sizeof(cfg))

	var pressed int32
	//nolint:gosec // Required unsafe.Pointer marshaling for the Win32 TASKDIALOGCONFIG syscall.
	ret, _, _ := procTaskDialogIndirect.Call(
		uintptr(unsafe.Pointer(&cfg)),
		uintptr(unsafe.Pointer(&pressed)),
		0,
		0,
	)
	return ret == 0 // S_OK == 0
}

func mustUTF16(s string) *uint16 {
	if s == "" {
		return nil
	}
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		return nil
	}
	return p
}

func infoDialog(title, instruction, content string) {
	if !taskDialog(title, instruction, content, tdInformationIcon, tdcbfOKButton) {
		legacyInfo(title, instruction, content)
	}
}

func errorDialog(title, instruction, content string) {
	if !taskDialog(title, instruction, content, tdErrorIcon, tdcbfOKButton) {
		legacyError(title, instruction, content)
	}
}
