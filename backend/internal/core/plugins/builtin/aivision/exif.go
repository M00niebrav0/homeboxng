package aivision

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// ExtractEXIFDateTime parses EXIF DateTimeOriginal from JPEG image bytes.
// Returns the timestamp or zero value if EXIF data is not found.
// Uses stdlib only (no external dependencies).
// Supports both little-endian (II/Intel) and big-endian (MM/Motorola) TIFF headers.
func ExtractEXIFDateTime(data []byte) time.Time {
	if len(data) < 12 {
		return time.Time{}
	}

	// Check JPEG SOI marker
	if data[0] != 0xFF || data[1] != 0xD8 {
		return time.Time{}
	}

	// Scan for APP1 (EXIF) marker: 0xFF 0xE1
	offset := 2
	for offset < len(data)-4 {
		if data[offset] != 0xFF {
			break
		}

		marker := data[offset+1]
		if marker == 0xE1 {
			// Found APP1 marker
			return parseAPP1(data[offset:])
		}

		// Skip to next marker
		if offset+3 >= len(data) {
			break
		}
		segLen := int(binary.BigEndian.Uint16(data[offset+2 : offset+4]))
		offset += 2 + segLen
	}

	return time.Time{}
}

func parseAPP1(data []byte) time.Time {
	if len(data) < 10 {
		return time.Time{}
	}

	segLen := int(binary.BigEndian.Uint16(data[2:4]))
	if segLen+2 > len(data) {
		return time.Time{}
	}

	// Check "Exif\x00\x00" header
	exifData := data[4 : 2+segLen]
	if len(exifData) < 8 || string(exifData[:4]) != "Exif" || exifData[4] != 0 || exifData[5] != 0 {
		return time.Time{}
	}

	tiffData := exifData[6:]
	if len(tiffData) < 8 {
		return time.Time{}
	}

	// Determine byte order
	var bo binary.ByteOrder
	switch string(tiffData[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return time.Time{}
	}

	// Verify TIFF magic number (42)
	if bo.Uint16(tiffData[2:4]) != 42 {
		return time.Time{}
	}

	// Get offset to first IFD
	ifdOffset := int(bo.Uint32(tiffData[4:8]))
	if ifdOffset >= len(tiffData) {
		return time.Time{}
	}

	// Search IFD0 for ExifIFD pointer (tag 0x8769)
	exifIFDOffset := findTag(tiffData, bo, ifdOffset, 0x8769)
	if exifIFDOffset == 0 {
		return time.Time{}
	}

	// Search ExifIFD for DateTimeOriginal (0x9003) and SubSecTimeOriginal (0x9291)
	dtOriginal := findTagString(tiffData, bo, int(exifIFDOffset), 0x9003)
	subSec := findTagString(tiffData, bo, int(exifIFDOffset), 0x9291)

	if dtOriginal == "" {
		return time.Time{}
	}

	// Parse "YYYY:MM:DD HH:MM:SS"
	t, err := time.Parse("2006:01:02 15:04:05", dtOriginal)
	if err != nil {
		return time.Time{}
	}

	// Add sub-second precision if available
	if subSec != "" {
		var frac float64
		if _, err := fmt.Sscanf("."+subSec, "%f", &frac); err == nil {
			nsec := int(math.Round(frac * 1e9))
			t = t.Add(time.Duration(nsec))
		}
	}

	return t
}

// findTag searches an IFD for a specific tag and returns its value as uint32.
func findTag(data []byte, bo binary.ByteOrder, ifdOffset int, targetTag uint16) uint32 {
	if ifdOffset+2 > len(data) {
		return 0
	}

	numEntries := int(bo.Uint16(data[ifdOffset : ifdOffset+2]))
	entryStart := ifdOffset + 2

	for i := 0; i < numEntries; i++ {
		eOffset := entryStart + i*12
		if eOffset+12 > len(data) {
			break
		}

		tag := bo.Uint16(data[eOffset : eOffset+2])
		if tag == targetTag {
			return bo.Uint32(data[eOffset+8 : eOffset+12])
		}
	}
	return 0
}

// findTagString searches an IFD for a tag and returns its value as a string.
func findTagString(data []byte, bo binary.ByteOrder, ifdOffset int, targetTag uint16) string {
	if ifdOffset+2 > len(data) {
		return ""
	}

	numEntries := int(bo.Uint16(data[ifdOffset : ifdOffset+2]))
	entryStart := ifdOffset + 2

	for i := 0; i < numEntries; i++ {
		eOffset := entryStart + i*12
		if eOffset+12 > len(data) {
			break
		}

		tag := bo.Uint16(data[eOffset : eOffset+2])
		if tag != targetTag {
			continue
		}

		dataType := bo.Uint16(data[eOffset+2 : eOffset+4])
		count := int(bo.Uint32(data[eOffset+4 : eOffset+8]))

		if dataType != 2 { // ASCII type
			return ""
		}

		var strData []byte
		if count <= 4 {
			strData = data[eOffset+8 : eOffset+8+count]
		} else {
			strOffset := int(bo.Uint32(data[eOffset+8 : eOffset+12]))
			if strOffset+count > len(data) {
				return ""
			}
			strData = data[strOffset : strOffset+count]
		}

		// Trim null terminator
		for len(strData) > 0 && strData[len(strData)-1] == 0 {
			strData = strData[:len(strData)-1]
		}
		return string(strData)
	}
	return ""
}
