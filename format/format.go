package format

import (
	"bytes"
	"fmt"
	"strconv"
)

const (
	KILOBYTE  uint64 = 1024
	MEGABYTE  uint64 = KILOBYTE * 1024
	GIGABYTE  uint64 = MEGABYTE * 1024
	TERRABYTE uint64 = GIGABYTE * 1024
)

// Integer ... Pretty print an integer value using commas, if it is large enough.
func Integer(n uint64, sep rune) string {
	s := strconv.FormatInt(int64(n), 10)

	startOffset := 0
	var buff bytes.Buffer

	if n < 0 {
		startOffset = 1
		buff.WriteByte('-')
	}

	l := len(s)

	commaIndex := 3 - ((l - startOffset) % 3)

	if commaIndex == 3 {
		commaIndex = 0
	}

	for i := startOffset; i < l; i++ {

		if commaIndex == 3 {
			buff.WriteRune(sep)
			commaIndex = 0
		}
		commaIndex++

		buff.WriteByte(s[i])
	}

	return buff.String()
}

// Number ... Print an integer value with a size suffix, keeping it within the provided restraints.
func Number(value uint64, width int32, prec int32, detailed bool, expandSizes bool) string {
	numString := ""
	if !expandSizes {
		widthFmt := ""
		if prec != 0 {
			widthFmt = fmt.Sprintf("%%%d.%df %%s", width, prec)
		} else {
			widthFmt = fmt.Sprintf("%%%dd %%s", width)
		}

		var sizeLabel string
		var sizeVal float64 = 0.0

		if value < KILOBYTE {
			sizeVal = float64(value)
			sizeLabel = "B"
		} else if value < MEGABYTE {
			sizeVal = (float64(value) / float64(KILOBYTE))
			sizeLabel = "KiB"
		} else if value < GIGABYTE {
			sizeVal = (float64(value) / float64(MEGABYTE))
			sizeLabel = "MiB"
		} else if value < TERRABYTE {
			sizeVal = (float64(value) / float64(GIGABYTE))
			sizeLabel = "GiB"
		} else {
			sizeVal = (float64(value) / float64(TERRABYTE))
			sizeLabel = "TiB"
		}

		if prec != 0 {
			numString = fmt.Sprintf(widthFmt, sizeVal, sizeLabel)
		} else {
			numString = fmt.Sprintf(widthFmt, int64(sizeVal), sizeLabel)
		}
	} else {
		widthFmt := fmt.Sprintf("%%%ds %%s", width)
		numString = fmt.Sprintf(widthFmt, Integer(value, ','), "B")
	}

	return numString
}
