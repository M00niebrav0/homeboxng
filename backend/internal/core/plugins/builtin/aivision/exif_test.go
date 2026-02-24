package aivision

import (
	"encoding/binary"
	"testing"
)

func TestExtractEXIFDateTime_NoJPEG(t *testing.T) {
	result := ExtractEXIFDateTime([]byte{0x00, 0x00, 0x00, 0x00})
	if !result.IsZero() {
		t.Error("expected zero time for non-JPEG data")
	}
}

func TestExtractEXIFDateTime_TooShort(t *testing.T) {
	result := ExtractEXIFDateTime([]byte{0xFF, 0xD8})
	if !result.IsZero() {
		t.Error("expected zero time for short data")
	}
}

func TestExtractEXIFDateTime_EmptyData(t *testing.T) {
	result := ExtractEXIFDateTime(nil)
	if !result.IsZero() {
		t.Error("expected zero time for nil data")
	}
}

func TestExtractEXIFDateTime_ValidJPEG_NoEXIF(t *testing.T) {
	// Valid JPEG with no APP1 marker
	data := []byte{
		0xFF, 0xD8, // SOI
		0xFF, 0xE0, // APP0 (JFIF) marker
		0x00, 0x10, // length 16
	}
	for i := 0; i < 14; i++ {
		data = append(data, 0x00)
	}
	data = append(data, 0xFF, 0xD9) // EOI

	result := ExtractEXIFDateTime(data)
	if !result.IsZero() {
		t.Error("expected zero time for JPEG without EXIF")
	}
}

// buildTestEXIF constructs a minimal JPEG+EXIF byte stream with a
// DateTimeOriginal tag. All string offsets are computed inline (no defer).
func buildTestEXIF(dateTime string, bo binary.ByteOrder) []byte {
	dtStr := dateTime + "\x00" // null-terminated

	// Build TIFF data from offset 0 (relative to TIFF header start)
	var tiff []byte

	// Byte order marker (offset 0-1)
	if bo == binary.LittleEndian {
		tiff = append(tiff, 'I', 'I')
	} else {
		tiff = append(tiff, 'M', 'M')
	}

	// TIFF magic 42 (offset 2-3)
	tmp2 := make([]byte, 2)
	bo.PutUint16(tmp2, 42)
	tiff = append(tiff, tmp2...)

	// IFD0 offset = 8 (right after this 8-byte header) (offset 4-7)
	tmp4 := make([]byte, 4)
	bo.PutUint32(tmp4, 8)
	tiff = append(tiff, tmp4...)

	// --- IFD0 at offset 8 ---
	// Number of entries: 1
	bo.PutUint16(tmp2, 1)
	tiff = append(tiff, tmp2...)

	// Entry: tag=0x8769 (ExifIFD pointer), type=LONG(4), count=1
	bo.PutUint16(tmp2, 0x8769)
	tiff = append(tiff, tmp2...)
	bo.PutUint16(tmp2, 4) // LONG
	tiff = append(tiff, tmp2...)
	bo.PutUint32(tmp4, 1) // count=1
	tiff = append(tiff, tmp4...)

	// Value: ExifIFD starts after IFD0 (IFD0: 2 + 12*1 + 4 = 18 bytes from offset 8, so offset 26)
	// But we need to account for the next-IFD pointer
	// After IFD0 entries: next IFD offset (4 bytes)
	// So ExifIFD offset = 8 + 2 + 12 + 4 = 26
	exifIFDOffset := uint32(8 + 2 + 12 + 4)
	bo.PutUint32(tmp4, exifIFDOffset)
	tiff = append(tiff, tmp4...)

	// Next IFD offset = 0 (no more IFDs)
	bo.PutUint32(tmp4, 0)
	tiff = append(tiff, tmp4...)

	// --- ExifIFD at offset 26 ---
	// Number of entries: 1 (DateTimeOriginal)
	bo.PutUint16(tmp2, 1)
	tiff = append(tiff, tmp2...)

	// Entry: tag=0x9003, type=ASCII(2), count=len(dtStr)
	bo.PutUint16(tmp2, 0x9003)
	tiff = append(tiff, tmp2...)
	bo.PutUint16(tmp2, 2) // ASCII
	tiff = append(tiff, tmp2...)
	bo.PutUint32(tmp4, uint32(len(dtStr)))
	tiff = append(tiff, tmp4...)

	// String data offset: comes after this IFD entry + next-IFD pointer
	// ExifIFD: 2 + 12*1 + 4 = 18 bytes total, so string starts at 26 + 18 = 44
	strOffset := exifIFDOffset + 2 + 12 + 4
	bo.PutUint32(tmp4, strOffset)
	tiff = append(tiff, tmp4...)

	// Next IFD offset = 0
	bo.PutUint32(tmp4, 0)
	tiff = append(tiff, tmp4...)

	// String data at offset 44
	tiff = append(tiff, []byte(dtStr)...)

	// Build EXIF payload: "Exif\x00\x00" + TIFF data
	var exif []byte
	exif = append(exif, 'E', 'x', 'i', 'f', 0x00, 0x00)
	exif = append(exif, tiff...)

	// APP1 segment length (includes its own 2 bytes)
	segLen := make([]byte, 2)
	binary.BigEndian.PutUint16(segLen, uint16(len(exif)+2))

	// Assemble JPEG: SOI + APP1 marker + segment length + exif data
	var result []byte
	result = append(result, 0xFF, 0xD8)       // SOI
	result = append(result, 0xFF, 0xE1)       // APP1 marker
	result = append(result, segLen...)          // segment length
	result = append(result, exif...)            // EXIF data

	return result
}

func TestExtractEXIFDateTime_LittleEndian(t *testing.T) {
	data := buildTestEXIF("2026:02:24 13:30:00", binary.LittleEndian)

	result := ExtractEXIFDateTime(data)
	if result.IsZero() {
		t.Fatal("expected non-zero time for valid EXIF (little-endian)")
	}

	if result.Year() != 2026 || result.Month() != 2 || result.Day() != 24 {
		t.Errorf("date = %v, want 2026-02-24", result)
	}
	if result.Hour() != 13 || result.Minute() != 30 {
		t.Errorf("time = %v, want 13:30:00", result)
	}
}

func TestExtractEXIFDateTime_BigEndian(t *testing.T) {
	data := buildTestEXIF("2025:12:25 08:00:00", binary.BigEndian)

	result := ExtractEXIFDateTime(data)
	if result.IsZero() {
		t.Fatal("expected non-zero time for valid EXIF (big-endian)")
	}

	if result.Year() != 2025 || result.Month() != 12 || result.Day() != 25 {
		t.Errorf("date = %v, want 2025-12-25", result)
	}
}

func TestFindTag_NotFound(t *testing.T) {
	data := make([]byte, 10)
	binary.LittleEndian.PutUint16(data[0:], 0) // 0 entries

	result := findTag(data, binary.LittleEndian, 0, 0x8769)
	if result != 0 {
		t.Errorf("expected 0 for not-found tag, got %d", result)
	}
}

func TestFindTag_BoundsCheck(t *testing.T) {
	data := make([]byte, 4)
	result := findTag(data, binary.LittleEndian, 100, 0x8769)
	if result != 0 {
		t.Error("expected 0 for out-of-bounds offset")
	}
}

func TestFindTagString_BoundsCheck(t *testing.T) {
	data := make([]byte, 4)
	result := findTagString(data, binary.LittleEndian, 100, 0x9003)
	if result != "" {
		t.Error("expected empty for out-of-bounds offset")
	}
}
