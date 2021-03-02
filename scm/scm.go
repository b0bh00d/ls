package scm

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	SCM_NONE = iota
	SCM_SVN
	SCM_HG
	SCM_GIT
)

const (
	STATUS_NONE = 0
	// lower nybble is basic status
	STATUS_DELETED = 1 << iota
	STATUS_RENAMED
	STATUS_ADDED
	STATUS_MODIFIED
)

// Entry ... This holds the SCM values for a single folder entry.
type Entry struct {
	Codes    string
	Bits     uint8
	Original string
}

// Status ... This holds the SCM status for all entries in the folder.
type Status struct {
	Manager  int
	MaxWidth int
	Entries  map[string]*Entry
	Deleted  map[string]*Entry
}

func detectScm(cwd string) int {
	target, err := filepath.Abs(cwd)
	if err != nil {
		log.Panic(err)
	}

	getScm := func(d string) int {
		s := fmt.Sprint(d, "\\.svn")
		if _, err := os.Stat(s); err == nil {
			return SCM_SVN
		}
		s = fmt.Sprint(d, "\\.hg")
		if _, err = os.Stat(s); err == nil {
			return SCM_HG
		}
		s = fmt.Sprint(d, "\\.git")
		if _, err = os.Stat(s); err == nil {
			return SCM_GIT
		}
		return SCM_NONE
	}

	for {
		scm := getScm(target)
		if scm != SCM_NONE {
			return scm
		}
		if len(target) == 3 && strings.HasSuffix(target, "\\") {
			break // we're at the top of the partition
		}
		target, err = filepath.Abs(fmt.Sprint(target, "\\.."))
		if err != nil {
			log.Panic(err)
		}
	}

	return SCM_NONE
}

func getSubversionStatus(status *Status) {
	cmd := exec.Command("svn", "status", "-q", ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Panic(err)
	}

	status.MaxWidth = 0

	var previousEntry string
	renamedMap := make(map[string]string)

	items := strings.Split(string(output), "\n")
	for _, line := range items {
		if len(line) != 0 {
			j := 7 // subversion codes span the first seven columns
			codes := strings.TrimRight(line[:j], " \n\r\t")
			if len(codes) > status.MaxWidth {
				status.MaxWidth = len(codes)
			}
			for line[j] == ' ' {
				j++
			}
			file := strings.TrimSpace(line[j:])
			pathItems := strings.Split(file, "\\")
			if len(pathItems) > 1 {
				file = pathItems[0] + "/"
			}

			var entry Entry
			entry.Codes = codes
			entry.Bits = STATUS_NONE

			// https://gotofritz.net/blog/svn-status-codes/
			if len(codes) == 0 {
				// this is what a rename looks like in Subversion:
				// A  +    bob.py
				//         > moved from reset.py
				// D       reset.py
				//         > moved to bob.py

				e := status.Entries[previousEntry]
				if (e.Bits & STATUS_ADDED) != 0 {
					e.Bits &^= STATUS_ADDED
					e.Bits |= STATUS_RENAMED

					file = file[13:] // strip "> moved from "

					// make note of the original file
					renamedMap[file] = previousEntry
				}
			} else {
				switch codes[0] {
				case 'D':
					entry.Bits = STATUS_NONE

					// is this an 'Original' file of a rename?
					var e Entry
					e.Codes = codes

					renamedTo, ok := renamedMap[file]
					if ok {
						e.Bits |= STATUS_RENAMED
						e.Original = file
						status.Deleted[renamedTo] = &e
					} else {
						e.Bits |= STATUS_DELETED
						status.Deleted[file] = &e
					}

				case 'R':
					entry.Bits |= STATUS_RENAMED

				case 'A':
					entry.Bits |= STATUS_ADDED
					previousEntry = file

				case 'M':
					entry.Bits |= STATUS_MODIFIED
				}
			}

			if entry.Bits != STATUS_NONE {
				status.Entries[file] = &entry
			}
		}
	}
}

func getMercurialStatus(status *Status) {
	cmd := exec.Command("hg", "status", "-C", "-q", ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Panic(err)
	}

	status.MaxWidth = 0

	var previousEntry string
	renamedMap := make(map[string]string)

	items := strings.Split(string(output), "\n")
	for _, line := range items {
		if len(line) != 0 {
			j := 0
			for line[j] != ' ' {
				j++
			}
			codes := strings.TrimRight(line[:j], " \n\r\t")
			if len(codes) > status.MaxWidth {
				status.MaxWidth = len(codes)
			}
			for line[j] == ' ' {
				j++
			}
			file := strings.TrimSpace(line[j:])
			pathItems := strings.Split(file, "\\")
			if len(pathItems) > 1 {
				file = pathItems[0] + "/"
			}

			var entry Entry
			entry.Codes = codes
			entry.Bits = STATUS_NONE

			// renames are a special case in Mercurial
			// https://stackoverflow.com/questions/2679488/showing-renames-in-hg-status
			if len(codes) == 0 {
				// the previous 'A'dd was actually a rename, so update it
				e := status.Entries[previousEntry]
				e.Bits &^= STATUS_ADDED
				e.Bits |= STATUS_RENAMED

				// make note of the original file
				renamedMap[file] = previousEntry
			} else {
				switch codes[0] {
				case 'R': // this is 'deleted' in HG
					// do some normalizing
					b := []byte(entry.Codes)
					b[0] = 'D'

					// is this an 'Original' file of a rename?
					var e Entry
					e.Codes = string(b)

					renamedTo, ok := renamedMap[file]
					if ok {
						e.Bits |= STATUS_RENAMED
						e.Original = file
						status.Deleted[renamedTo] = &e
					} else {
						e.Bits |= STATUS_DELETED
						status.Deleted[file] = &e
					}

				case 'A': // a blank status line following an add ('A') is a rename in HG
					entry.Bits |= STATUS_ADDED
					previousEntry = file

				case 'M':
					entry.Bits |= STATUS_MODIFIED
				}
			}

			if entry.Bits != STATUS_NONE {
				status.Entries[file] = &entry
			}
		}
	}
}

func getGitStatus(status *Status) {
	cmd := exec.Command("git", "status", "--porcelain", "-uno", ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Panic(err)
	}

	status.MaxWidth = 0

	items := strings.Split(string(output), "\n")
	for _, line := range items {
		if len(line) != 0 {
			j := 2 // git codes in porcelain mode span the first two columns
			codes := strings.TrimRight(line[:j], " \n\r\t")
			if len(codes) > status.MaxWidth {
				status.MaxWidth = len(codes)
			}
			for line[j] == ' ' {
				j++
			}
			file := strings.TrimSpace(line[j:])
			pathItems := strings.Split(file, "\\")
			if len(pathItems) > 1 {
				file = pathItems[0] + "/"
			}

			// https://git-scm.com/docs/git-status

			var entry Entry
			entry.Codes = codes
			entry.Bits = STATUS_NONE

			if len(codes) != 0 {
				switch {
				case codes[0] == 'D':
					var e Entry
					e.Codes = codes
					e.Bits |= STATUS_DELETED
					status.Deleted[file] = &e

				case codes[0] == 'R':
					// 'file' looks like "<oldname> -> <newname>"
					index := strings.Index(file, " -> ")
					original := file[:index]
					file = file[index+4:]

					entry.Bits |= STATUS_RENAMED

					// is this an 'Original' file of a rename?
					var e Entry
					e.Codes = codes

					e.Bits |= STATUS_RENAMED
					e.Original = original
					status.Deleted[file] = &e

				case codes[0] == 'A':
					entry.Bits |= STATUS_ADDED

				// git's codes are a little more complicated because it has
				// the staging area, and a file can be modified in the working
				// tree or the staging area and the status will be in different
				// columns depending.
				case codes[0] == 'M' || (len(codes) > 1 && codes[1] == 'M'):
					entry.Bits |= STATUS_MODIFIED
					// messes up coloring!
					// if len(codes) > 1 {
					// 	// substitute a lower-case code to show that it is not
					// 	// yet staged for committing
					// 	b := []byte(codes)
					// 	b[1] = 'm'
					// 	entry.Codes = string(b)
					// }
				}
			}

			if entry.Bits != STATUS_NONE {
				status.Entries[file] = &entry
			}
		}
	}
}

// GetScmStatus ... This is a single entry point for detecting the presence of one of the
// three most popular SCM system.  If one is found, then the status of the current folder within
// context of that manager will be returned to the caller.
func GetScmStatus(cwd string) Status {
	var status Status
	status.Entries = make(map[string]*Entry)
	status.Deleted = make(map[string]*Entry)
	status.Manager = detectScm(cwd)
	if status.Manager != SCM_NONE {
		switch status.Manager {
		case SCM_SVN:
			getSubversionStatus(&status)
		case SCM_HG:
			getMercurialStatus(&status)
		case SCM_GIT:
			getGitStatus(&status)
		}
	}
	return status
	// return scm_type, status, codes_width
}
