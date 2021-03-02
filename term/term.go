package term

import (
	"github.com/nathan-fiscaletti/consolesize-go"
	"golang.org/x/sys/windows"
)

var dllTerminal *windows.DLL = nil

// GetDimensions ... Returns the dimensions of the current console window.
func GetDimensions() (int, int) {
	cols, rows := consolesize.GetConsoleSize()
	return rows, cols
}

// EnableColor ... This function uses shared library functions to see if ANSI coloring
// can be used in the current console.  It will attempt to enable coloring if it isn't
// already, and will report the results.
func EnableColor() bool {
	result := false

	if dllTerminal == nil {
		dll, err := windows.LoadDLL("terminal.dll")
		if err == nil {
			dllTerminal = dll
		}
	}

	if dllTerminal != nil {
		// if we have the DLL, we MUST have the entry points
		procHasColorSupport := dllTerminal.MustFindProc("HasColorSupport")
		procIsColorSupportEnabled := dllTerminal.MustFindProc("IsColorSupportEnabled")
		procEnableColorSupport := dllTerminal.MustFindProc("EnableColorSupport")

		r, _, _ := procHasColorSupport.Call()
		if r == 1 {
			r, _, _ := procIsColorSupportEnabled.Call()
			if r == 0 {
				enabled, _, _ := procEnableColorSupport.Call(1)
				if enabled == 1 {
					result = true
				}
			} else {
				result = true
			}
		}
	}

	return result
}
