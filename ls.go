package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/fatih/color"
	"golang.org/x/sys/windows"

	"github.com/b0bh00d/ls/format"
	"github.com/b0bh00d/ls/meta"
	"github.com/b0bh00d/ls/scm"
	"github.com/b0bh00d/ls/term"
)

type entryData struct {
	file    string
	modtime time.Time
	size    uint64
	sizeDsp float64
	sizeFmt string
	stats   string
	symlink string
	isDir   bool
}

var totalBytes uint64 = 0

var dllKernel32 *windows.DLL = nil
var procGetDiskFreeSpaceW *windows.Proc = nil

type partitionInfo struct {
	sectorsPerCluster     uint64
	bytesPerSector        uint64
	numberOfFreeClusters  uint64
	totalNumberOfClusters uint64
	totalBytes            uint64
	bytesFree             uint64
}

func getPartInfo(path string) *partitionInfo {
	var sectorsPerCluster uint32 = 0
	var bytesPerSector uint32 = 0
	var numberOfFreeClusters uint32 = 0
	var totalNumberOfClusters uint32 = 0

	result, _, err := procGetDiskFreeSpaceW.Call(uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(path))),
		uintptr(unsafe.Pointer(&sectorsPerCluster)),
		uintptr(unsafe.Pointer(&bytesPerSector)),
		uintptr(unsafe.Pointer(&numberOfFreeClusters)),
		uintptr(unsafe.Pointer(&totalNumberOfClusters)))

	if result != 1 {
		log.Panic(err)
	}

	clusterSize := uint64(sectorsPerCluster * bytesPerSector)
	totalBytes := uint64(totalNumberOfClusters) * clusterSize
	bytesFree := totalBytes - uint64(numberOfFreeClusters)*clusterSize

	partInfo := partitionInfo{uint64(sectorsPerCluster), uint64(bytesPerSector), uint64(numberOfFreeClusters), uint64(totalNumberOfClusters), totalBytes, bytesFree}

	return &partInfo
}

// https://stackoverflow.com/questions/14668850/list-directory-in-go
func readDir(root string) []string {
	var files []string
	f, err := os.Open(root)
	if err != nil {
		log.Panic(err)
	}
	fileInfo, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		log.Panic(err)
	}

	for _, file := range fileInfo {
		files = append(files, file.Name())
	}
	return files
}

func resolveReparsePoint(file string) string {
	fi, err := os.Lstat(file)
	if err != nil {
		log.Panic(err)
	}

	if fi.Mode()&os.ModeSymlink == 0 {
		return file
	}

	target, err := os.Readlink(file)
	if err != nil {
		log.Panic(err)
	}

	if !filepath.IsAbs(target) {
		target, err = filepath.Abs(target)
		if err != nil {
			log.Fatal(err)
		}
	}

	return target
}

// smoe attributes aren't defined in syscall, so I"ve had to define them here
const FILE_ATTRIBUTE_COMPRESSED uint32 = 2048
const FILE_ATTRIBUTE_ENCRYPTED uint32 = 16384
const FILE_ATTRIBUTE_SPARSE_FILE uint32 = 512

func processStats(file string) (string, string) {
	flags := []string{"-", "-", "-", "-", "-", "-", "-", "-"}

	ptr, err := syscall.UTF16PtrFromString(file)
	if err != nil {
		log.Panic(err)
	}

	// https://golang.hotexamples.com/examples/syscall/-/GetFileAttributes/golang-getfileattributes-function-examples.html
	attr, err := syscall.GetFileAttributes(ptr)
	if err != nil {
		log.Panic(err)
	}

	if attr&syscall.FILE_ATTRIBUTE_DIRECTORY != 0 {
		file = fmt.Sprint(file, "/")
	}

	if attr&syscall.FILE_ATTRIBUTE_READONLY != 0 {
		flags[0] = "r"
	}
	if attr&syscall.FILE_ATTRIBUTE_ARCHIVE != 0 {
		flags[1] = "a"
	}
	if attr&syscall.FILE_ATTRIBUTE_HIDDEN != 0 {
		flags[2] = "h"
	}
	if attr&syscall.FILE_ATTRIBUTE_SYSTEM != 0 {
		flags[3] = "s"
	}
	if attr&FILE_ATTRIBUTE_COMPRESSED != 0 {
		flags[4] = "c"
	}
	if attr&FILE_ATTRIBUTE_ENCRYPTED != 0 {
		flags[5] = "e"
	}
	if attr&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		flags[6] = "S"
	}
	if attr&FILE_ATTRIBUTE_SPARSE_FILE != 0 {
		flags[7] = "p"
	}

	return file, strings.Join(flags, "")
}

func processFile(file string) entryData {
	fi, err := os.Stat(file)
	if err != nil {
		log.Fatal(err)
	}

	t := fi.ModTime()
	file, stats := processStats(file)
	var symlinkTarget string
	if stats[6] == 'S' {
		// this is a reparse point (a.k.a. symlink)
		symlinkTarget = resolveReparsePoint(file)
	}

	// var size_str string = fmt.Sprintf("%7s    ", " ")
	var sizeFmt string
	if lsConfigData.compactSizes {
		sizeFmt = fmt.Sprintf("%7s    ", " ")
	} else {
		sizeFmt = fmt.Sprintf("%15s    ", " ")
	}
	var sizeVal = 0.0

	s := uint64(0)

	if !strings.HasSuffix(file, "/") {
		s = uint64(fi.Size())
		totalBytes += s

		if lsConfigData.compactSizes {
			sizeFmt = fmt.Sprintf("%7s    ", " ")
			if s < format.KILOBYTE {
				sizeVal = float64(s)
				sizeFmt = "%7.2f B  "
			} else if s < format.MEGABYTE {
				sizeVal = (float64(s) / float64(format.KILOBYTE))
				sizeFmt = "%7.2f KiB"
			} else if s < format.GIGABYTE {
				sizeVal = (float64(s) / float64(format.MEGABYTE))
				sizeFmt = "%7.2f MiB"
			} else if s < format.TERRABYTE {
				sizeVal = (float64(s) / float64(format.GIGABYTE))
				sizeFmt = "%7.2f GiB"
			} else {
				sizeVal = (float64(s) / float64(format.TERRABYTE))
				sizeFmt = "%7.2f TiB"
			}
		} else {
			sizeVal = float64(s)
			sizeFmt = "%17s B"
		}
	}

	// https://flaviocopes.com/go-date-time-format/
	// timestamp := t.Format("01/02/06 15:04:05")

	return entryData{file: file, modtime: t, size: s, sizeDsp: sizeVal, sizeFmt: sizeFmt, stats: stats, symlink: symlinkTarget, isDir: fi.IsDir()}
}

func quickSort(a []entryData, ascending bool) []entryData {
	if len(a) < 2 {
		return a
	}

	left, right := 0, len(a)-1

	pivot := rand.Int() % len(a)

	a[pivot], a[right] = a[right], a[pivot]

	for i := range a {
		swap := false
		if ascending {
			swap = a[i].modtime.Unix() < a[right].modtime.Unix()
		} else {
			swap = a[i].modtime.Unix() > a[right].modtime.Unix()
		}
		if swap {
			a[left], a[i] = a[i], a[left]
			left++
		}
	}

	a[left], a[right] = a[right], a[left]

	quickSort(a[:left], ascending)
	quickSort(a[left+1:], ascending)

	return a
}

func colorizeCodes(codes string) string {
	newString := ""
	for i := range codes {
		code := string(codes[i])
		if code != " " {
			color, ok := lsConfigData.coloring[code]
			if ok {
				newString += color.Sprint(code)
			} else {
				newString += code
			}
		} else {
			newString += code
		}
	}

	return newString
}

func main() {
	rows, cols := term.GetDimensions()

	dllKernel32 = windows.MustLoadDLL("kernel32.dll")
	procGetDiskFreeSpaceW = dllKernel32.MustFindProc("GetDiskFreeSpaceW")

	dllMsvcrt := windows.MustLoadDLL("msvcrt.dll")
	procGetch := dllMsvcrt.MustFindProc("_getch")

	color.NoColor = !term.EnableColor()

	loadConfig()
	parseCommandLine()

	var tasks = map[string][]string{}

	for _, val := range flag.Args() {
		if stat, err := os.Stat(val); err == nil && stat.IsDir() {
			// path is a directory
			tasks[val] = []string{"*"}
		} else {
			// handle it as a file pattern
			d := filepath.Dir(val)
			f := val
			if d != "." {
				f = val[len(d)+1:]
			}
			_, ok := tasks[d]
			if ok {
				// already has files_to_process
				tasks[d] = append(tasks[d], f)
			} else {
				tasks[d] = []string{f}
			}
		}
	}

	if len(tasks) == 0 {
		tasks["."] = []string{"*"}
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// lambdas
	lElideFunc := func(filename string, remaining int) string {
		if lsConfigData.elideLongNames {
			line := ""
			if len(filename) > remaining {
				left := filename[:len(filename)/2]
				right := filename[len(left):]
				newline := left + right
				count := 0
				for len(newline) > remaining {
					if (count & 1) == 0 {
						right = right[1:]
					} else {
						left = left[:len(left)-1]
					}
					newline = left + right
					count++
				}
				line = fmt.Sprint(left, "...", right)
				return line
			}
		}
		return filename
	}

	lProcessFile := func(entry entryData, cwd string, scmStatus *scm.Status) string {
		entrySize := ""
		if lsConfigData.compactSizes {
			entrySize = fmt.Sprintf(entry.sizeFmt, entry.sizeDsp)
			entrySize += strings.Repeat(" ", 11-len(entrySize))
		} else {
			entrySize = fmt.Sprintf(entry.sizeFmt, format.Integer(entry.size, ','))
			entrySize += strings.Repeat(" ", 19-len(entrySize))
		}

		scmLine := ""
		scmRename := ""

		if len(scmStatus.Entries) != 0 || len(scmStatus.Deleted) != 0 {
			scmLine = strings.Repeat(" ", scmStatus.MaxWidth)
			scmEntry, ok := scmStatus.Entries[entry.file]
			if ok {
				scmLine = scmEntry.Codes
				scmLine += strings.Repeat(" ", scmStatus.MaxWidth-len(scmEntry.Codes))
				scmLine = colorizeCodes(scmLine)
				e, ok := scmStatus.Deleted[entry.file]
				if ok {
					scmRename = e.Original
				}
			}
			scmLine += " "
		}

		line := fmt.Sprint(entry.modtime.Format("01/02/06 15:04:05"), " ", entrySize, " ", entry.stats, " ")
		remaining := cols - (len(line) + len(scmLine)) - 4

		lineToElide := entry.file
		if len(scmRename) != 0 {
			lineToElide = fmt.Sprintf("%s [née %s]", entry.file, scmRename)
		}
		line += fmt.Sprint(lElideFunc(lineToElide, remaining))

		// retrieve file metadata based on priority
		metacolor := "description"
		metadata := ""
		if !lsConfigData.hideMetaData {
			metadata = meta.Retrieve(entry.file, cwd)
		}
		if len(metadata) == 0 {
			if len(metadata) == 0 {
				metadata = ""
				if !lsConfigData.hideLinks {
					metadata = entry.symlink
				}
				if len(metadata) != 0 {
					metacolor = "symlink"
					metadata = fmt.Sprintf("@%s", metadata)
				}
			}
		}

		metaDataLength := len(metadata)
		if metaDataLength != 0 {
			needed := len(line) + metaDataLength + 4
			if needed < cols {
				colsLeft := cols - len(line) - len(metadata) - 4
				line += " "
				line += strings.Repeat("-", colsLeft)
				line += "> "
			} else {
				metaDataLength = 0
			}
		}

		var fileColor *color.Color = nil

		ext := filepath.Ext(entry.file)
		if len(ext) != 0 {
			color, ok := lsConfigData.coloring[strings.ToLower(ext)[1:]]
			if ok {
				fileColor = color
			}
		}

		if fileColor != nil {
			line = fileColor.Sprint(line)
		}

		line = fmt.Sprintf("%s%s", scmLine, line)

		if metaDataLength != 0 {
			line += lsConfigData.coloring[metacolor].Sprint(metadata)
		}

		return line
	}

	lProcessDir := func(entry entryData, cwd string, scmStatus *scm.Status) string {
		scmLine := ""
		scmRename := ""

		if len(scmStatus.Entries) != 0 || len(scmStatus.Deleted) != 0 {
			scmLine = strings.Repeat(" ", scmStatus.MaxWidth)
			scmEntry, ok := scmStatus.Entries[entry.file]
			if ok {
				scmLine = scmEntry.Codes
				scmLine += strings.Repeat(" ", scmStatus.MaxWidth-len(scmEntry.Codes))
				scmLine = colorizeCodes(scmLine)
				e, ok := scmStatus.Deleted[entry.file]
				if ok {
					scmRename = e.Original
				}
			}
			scmLine += " "
		}

		line := fmt.Sprint(entry.modtime.Format("01/02/06 15:04:05"), " ", entry.sizeFmt, " ", entry.stats, " ")
		remaining := cols - (len(line) + len(scmLine)) - 4

		lineToElide := entry.file
		if len(scmRename) != 0 {
			lineToElide = fmt.Sprintf("%s [née %s]", entry.file, scmRename)
		}
		line += fmt.Sprint(lElideFunc(lineToElide, remaining))

		// retrieve directory metadata based on priority
		metacolor := "description"
		metadata := ""
		if !lsConfigData.hideMetaData {
			metadata = meta.Retrieve(entry.file, cwd)
		}
		if len(metadata) == 0 {
			metadata = ""
			if !lsConfigData.hideLinks {
				metadata = entry.symlink
			}
			if len(metadata) != 0 {
				metacolor = "symlink"
				metadata = fmt.Sprintf("@%s", metadata)
			}
		}

		metaDataLength := len(metadata)
		if metaDataLength != 0 {
			needed := (len(line) + len(scmLine)) + metaDataLength + 4
			if needed < cols {
				colsLeft := cols - (len(line) + len(scmLine)) - len(metadata) - 4
				line += " "
				line += strings.Repeat("-", colsLeft)
				line += "> "
			} else {
				metaDataLength = 0
			}
		}

		line = fmt.Sprintf("%s%s", scmLine, lsConfigData.coloring["directories"].Sprint(line))

		if metaDataLength != 0 {
			line += lsConfigData.coloring[metacolor].Sprint(metadata)
		}

		return line
	}

	linesPrinted := 0
	lPrintLine := func(line string) {
		fmt.Println(line)
		linesPrinted++
		if lsConfigData.autoMore {
			if linesPrinted == (rows - 1) {
				fmt.Print("Press SPACE key to continue...\r")
				for {
					result, _, _ := procGetch.Call()
					if result == ' ' {
						break
					} else if result == 3 {
						fmt.Print("                               ")
						os.Exit(0)
					}
				}
				linesPrinted = 0
			}
		}
	}
	firstListing := true

	for key, patterns := range tasks {
		if !firstListing {
			fmt.Printf("\n|%s|\n\n", strings.Repeat("-", cols-3))
			linesPrinted += 3
		}

		if key != "." && key != cwd {
			os.Chdir(key)
		}

		cwd, err := filepath.Abs(".")
		if err != nil {
			log.Fatal(err)
		}
		if cwd[len(cwd)-1] == '\\' {
			cwd = cwd[:len(cwd)-1]
		}

		partInfo := getPartInfo(cwd)

		// is this a managed folder?
		scmStatus := scm.GetScmStatus(cwd)

		maxLineLength := 0
		allocatedBytes := uint64(0)

		var fileEntries []entryData
		var dirEntries []entryData

		files := readDir(".")

		// categorize and file each entry
		for i := range files {
			matched := false
			for j := range patterns {
				succ, err := filepath.Match(patterns[j], files[i])
				if err != nil {
					log.Panic(err)
				}
				if succ {
					matched = true
					break
				}
			}

			if !matched {
				continue
			}

			entry := processFile(files[i])

			if lsConfigData.hideHidden && entry.stats[2] == 'h' {
				continue
			}
			if lsConfigData.hideSystem && entry.stats[3] == 's' {
				continue
			}

			var entrySize string
			if entry.isDir {
				entrySize = entry.sizeFmt
			} else {
				entrySize = fmt.Sprint(entry.sizeFmt, entry.sizeDsp)
				if partInfo.bytesPerSector > 0 {
					allocatedBytes += partInfo.bytesPerSector * (entry.size / partInfo.bytesPerSector)
					if entry.size%partInfo.bytesPerSector != 0 {
						allocatedBytes += partInfo.bytesPerSector
					}
				}
			}

			line := ""
			if scmStatus.Manager != scm.SCM_NONE {
				line = strings.Repeat(" ", scmStatus.MaxWidth)
				scmEntry, ok := scmStatus.Entries[entry.file]
				if ok {
					line += fmt.Sprintf("%s ", scmEntry.Codes)
				}
			}

			line += fmt.Sprint(entry.modtime.Format("01/02/06 15:04:05"), " ", entrySize, " ", entry.stats, " ", entry.file)
			l := len(line)
			if l > maxLineLength {
				maxLineLength = l
			}

			if entry.isDir {
				dirEntries = append(dirEntries, entry)
			} else {
				fileEntries = append(fileEntries, entry)
			}
		}

		patternsDisp := strings.Join(patterns, ",")
		if strings.Contains(patternsDisp, ",") {
			patternsDisp = fmt.Sprintf("[%s]", patternsDisp)
		}
		lPrintLine(fmt.Sprintf(" Directory of %s\\%s", cwd, patternsDisp))
		lPrintLine("")

		finalLines := []string{}

		if lsConfigData.sortAscending || lsConfigData.sortDescending {
			entries := append(dirEntries, fileEntries...)
			ascending := true
			if !lsConfigData.sortAscending {
				ascending = false
			}
			entries = quickSort(entries, ascending)
			for i := range entries {
				if entries[i].isDir {
					finalLines = append(finalLines, lProcessDir(entries[i], cwd, &scmStatus))
				} else {
					finalLines = append(finalLines, lProcessFile(entries[i], cwd, &scmStatus))
				}
			}
		} else {
			if !lsConfigData.fileFirst {
				for i := range dirEntries {
					finalLines = append(finalLines, lProcessDir(dirEntries[i], cwd, &scmStatus))
				}
				for i := range fileEntries {
					finalLines = append(finalLines, lProcessFile(fileEntries[i], cwd, &scmStatus))
				}
			} else {
				for i := range fileEntries {
					finalLines = append(finalLines, lProcessFile(fileEntries[i], cwd, &scmStatus))
				}
				for i := range dirEntries {
					finalLines = append(finalLines, lProcessDir(dirEntries[i], cwd, &scmStatus))
				}
			}

			// pick up the case where a file under SCM management has been deleted (and won't
			// appear in the normal directory listing)
			if scmStatus.Manager != scm.SCM_NONE {
				if len(scmStatus.Deleted) != 0 {
					firstLine := true

					for key, e := range scmStatus.Deleted {
						scmLine := ""
						if (e.Bits & scm.STATUS_DELETED) != 0 {
							if firstLine {
								scmLine = strings.Repeat(" ", scmStatus.MaxWidth)
								scmLine += " -----------------"
								finalLines = append(finalLines, scmLine)

								firstLine = false
							}
							// scmEntry := scmStatus.Entries[key]
							scmLine = colorizeCodes(e.Codes)
							if len(e.Codes) < scmStatus.MaxWidth {
								scmLine += strings.Repeat(" ", scmStatus.MaxWidth-len(e.Codes))
							}
							scmLine += " "
							scmLine += key
							finalLines = append(finalLines, scmLine)
						}
					}
				}
			}
		}

		for _, val := range finalLines {
			lPrintLine(val)
		}

		lPrintLine("")

		if len(fileEntries) != 0 || len(dirEntries) != 0 {
			prefix := fmt.Sprintf("%s in", format.Number(totalBytes, 20, 2, false, !lsConfigData.compactSizes))
			fileData := ""
			dirData := ""

			if len(fileEntries) != 0 {
				fileData = fmt.Sprintf("%d file", len(fileEntries))
				if len(fileEntries) > 1 {
					fileData += "s"
				}
			}
			if len(dirEntries) != 0 {
				dirData = fmt.Sprintf("%d dir", len(dirEntries))
				if len(dirEntries) > 1 {
					dirData += "s"
				}
			}
			fmt.Print(prefix)
			if len(fileData) != 0 {
				fmt.Printf(" %s", fileData)
			}
			if len(dirEntries) != 0 {
				if len(fileData) != 0 {
					fmt.Print(" and")
				}
				fmt.Printf(" %s", dirData)
			}
			if len(fileData) != 0 && partInfo.bytesPerSector > 0 {
				fmt.Printf(" / %s allocated (", format.Number(allocatedBytes, 0, 2, false, !lsConfigData.compactSizes))
				lsConfigData.coloring["description"].Printf("%s slack", format.Number(allocatedBytes-totalBytes, 0, 2, false, !lsConfigData.compactSizes))
				fmt.Print(")")
			}
			lPrintLine("")
		} else {
			lPrintLine(fmt.Sprintf("%20s0 bytes in 0 files and 0 dirs", " "))
		}

		p := (float64(partInfo.bytesFree) / float64(partInfo.totalBytes)) * 100.0
		lPrintLine(fmt.Sprintf("%s free of %s (%.1f%%)", format.Number(partInfo.bytesFree, 20, 2, false, !lsConfigData.compactSizes), format.Number(partInfo.totalBytes, 0, 2, false, !lsConfigData.compactSizes), p))

		if key != "." {
			os.Chdir(cwd)
		}

		firstListing = false
	}
}
