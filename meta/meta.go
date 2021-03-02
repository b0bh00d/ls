package meta

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/sys/windows"
)

var dllMeta *windows.DLL = nil
var procMeta *windows.Proc = nil

// https://stackoverflow.com/questions/15783830/how-to-read-utf16-text-file-to-string-in-golang
func utf16BytesToString(b []byte) string {
	utf := make([]uint16, (len(b)+(2-1))/2)
	for i := 0; i+(2-1) < len(b); i += 2 {
		// I'm hard-coding LittleEndian because this is intended to run on a Windows machine
		utf[i/2] = binary.LittleEndian.Uint16(b[i:])
	}
	if len(b)/2 < len(utf) {
		utf[len(utf)-1] = utf8.RuneError
	}
	return string(utf16.Decode(utf))
}

// See if the file (or folder) has a Directory Opus metadata description assigned
func getMetadata(filename string) string {
	// Directory Opus stores comments (file and directory) in NTFS Alternate Data Streams (ADS)

	if dllMeta == nil {
		dll, err := windows.LoadDLL("metadata.dll")
		if err == nil {
			dllMeta = dll
			proc, err := dllMeta.FindProc("retrieve_metadata")
			if err == nil {
				procMeta = proc
			}
		}
	}

	result := ""
	if procMeta != nil {
		if strings.HasSuffix(filename, "/") {
			filename = filename[:len(filename)-1]
		}

		const DOPUS_BUFFER_SIZE = 2048
		buffer := make([]byte, DOPUS_BUFFER_SIZE)
		var pBuffer *byte
		pBuffer = &buffer[0]

		len, _, _ := procMeta.Call(
			uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(filename))),
			uintptr(unsafe.Pointer(pBuffer)),
			uintptr(DOPUS_BUFFER_SIZE))

		if len != 0 {
			r := utf16BytesToString(buffer)
			result = r[:len]
		}
	}

	return result
}

// Total Commander descript.ion file (honestly, I don't know if
// this file is still in use in 2021, but it's here just in case)
func getDescriptions(descriptions map[string]string) {
	file, err := os.Open("descript.ion")
	if err == nil {
		defer file.Close()

		lineNo := 1

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			filename := ""
			text := ""
			line := scanner.Text()
			if line[0] == '"' || line[0] == '\'' {
				filename = ""
				startChar := line[0]
				index := 1
				for line[index] != startChar {
					filename = fmt.Sprintf("%s%s", filename, line[index])
					index++
				}
				for line[index] == ' ' {
					index++
				}
				text = line[index:]
			} else {
				index := strings.Index(line, " ")
				filename = line[:index]
				text = line[index+1:]
			}

			descriptions[filename] = text
			lineNo++
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}
}

var currentWorkingDir string
var descriptions map[string]string

// Retrieve ... This function will check for several types of metadata on the indicated
// file, and return any it finds.  If there are more than one found, then they are
// arbitrarily prioritized.
func Retrieve(filename string, cdir string) string {
	meta := getMetadata(filename)
	if len(meta) == 0 {
		// cache the descriptions until the cwd changes
		if currentWorkingDir != cdir {
			descriptions = make(map[string]string)
			getDescriptions(descriptions)
			currentWorkingDir = cdir
		}
		value, ok := descriptions[filename]
		if ok {
			meta = value
		}
	}

	return meta
}
