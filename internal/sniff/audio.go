package sniff

import (
	"bytes"
	"encoding/binary"

	"golang.org/x/exp/slices"
)

type Sniffer interface {
	Sniff(header []byte) bool
}

var audioExtensions = map[string]Sniffer{
	// ref: https://mimesniff.spec.whatwg.org
	".mp3": &mp3Sniffer{}, // Enhanced MP3 detection with and without ID3v2 tag
	".ogg": prefixSniffer("OggS"),
	".wav": prefixSniffer("RIFF"),

	// ref: https://www.loc.gov/preservation/digital/formats/fdd/fdd000027.shtml
	".wma": prefixSniffer{
		0x30, 0x26, 0xb2, 0x75, 0x8e, 0x66, 0xcf, 0x11,
		0xa6, 0xd9, 0x00, 0xaa, 0x00, 0x62, 0xce, 0x6c,
	},

	// ref: https://www.garykessler.net/library/file_sigs.html
	".m4a": m4aSniffer{},    // MPEG-4 container, Apple Lossless Audio Codec
	".mp4": &mpeg4Sniffer{}, // MPEG-4 container, other fallback

	".flac": prefixSniffer("fLaC"), // ref: https://xiph.org/flac/format.html
	".dff":  prefixSniffer("FRM8"), // DSDIFF, ref: https://www.sonicstudio.com/pdf/dsd/DSDIFF_1.5_Spec.pdf

}

// AudioExtension sniffs the known audio types, and returns the file extension.
// header is recommended to at least 16 bytes.
func AudioExtension(header []byte) (string, bool) {
	// Check specific formats first to avoid false positives
	// Order matters: more specific formats should be checked before generic ones

	// Check for formats with clear magic headers first
	if bytes.HasPrefix(header, []byte("OggS")) {
		return ".ogg", true
	}
	if bytes.HasPrefix(header, []byte("fLaC")) {
		return ".flac", true
	}
	if bytes.HasPrefix(header, []byte("RIFF")) {
		return ".wav", true
	}
	if bytes.HasPrefix(header, []byte("FRM8")) {
		return ".dff", true
	}

	// Check WMA format
	wmaHeader := []byte{0x30, 0x26, 0xb2, 0x75, 0x8e, 0x66, 0xcf, 0x11, 0xa6, 0xd9, 0x00, 0xaa, 0x00, 0x62, 0xce, 0x6c}
	if bytes.HasPrefix(header, wmaHeader) {
		return ".wma", true
	}

	// Check M4A format
	m4a := m4aSniffer{}
	if m4a.Sniff(header) {
		return ".m4a", true
	}

	// Check MP4 format
	mp4 := &mpeg4Sniffer{}
	if mp4.Sniff(header) {
		return ".mp4", true
	}

	// Check MP3 last (as it can have false positives)
	if (&mp3Sniffer{}).Sniff(header) {
		return ".mp3", true
	}

	return "", false
}

// AudioExtensionWithFallback is equivalent to AudioExtension, but returns fallback
// most likely to use .mp3 as fallback, because mp3 files may not have ID3v2 tag.
func AudioExtensionWithFallback(header []byte, fallback string) string {
	ext, ok := AudioExtension(header)
	if !ok {
		return fallback
	}
	return ext
}

// AudioExtensionWithSmartFallback is like AudioExtensionWithFallback, but uses
// intelligent fallback based on input file extension when format sniffing fails.
func AudioExtensionWithSmartFallback(header []byte, inputExt string) string {
	ext, ok := AudioExtension(header)
	if !ok {
		// Use smart fallback based on input file extension
		return getSmartFallback(inputExt)
	}
	return ext
}

// getSmartFallback returns the expected output format based on input file extension
func getSmartFallback(inputExt string) string {
	switch inputExt {
	case ".mgg", ".mgg0", ".mgg1", ".mgga", ".mggh", ".mggl", ".mggm":
		return ".ogg"
	case ".mflac", ".mflac0", ".mflac1", ".mflaca", ".mflach", ".mflacl", ".mflacm":
		return ".flac"
	case ".qmcflac":
		return ".flac"
	case ".qmcogg":
		return ".ogg"
	default:
		return ".mp3" // default fallback
	}
}

type prefixSniffer []byte

func (s prefixSniffer) Sniff(header []byte) bool {
	return bytes.HasPrefix(header, s)
}

type m4aSniffer struct{}

func (m4aSniffer) Sniff(header []byte) bool {
	box := readMpeg4FtypBox(header)
	if box == nil {
		return false
	}

	return box.majorBrand == "M4A " || slices.Contains(box.compatibleBrands, "M4A ")
}

type mpeg4Sniffer struct{}

func (s *mpeg4Sniffer) Sniff(header []byte) bool {
	return readMpeg4FtypBox(header) != nil
}

type mpeg4FtpyBox struct {
	majorBrand       string
	minorVersion     uint32
	compatibleBrands []string
}

func readMpeg4FtypBox(header []byte) *mpeg4FtpyBox {
	if (len(header) < 8) || !bytes.Equal([]byte("ftyp"), header[4:8]) {
		return nil // not a valid ftyp box
	}

	size := binary.BigEndian.Uint32(header[0:4]) // size
	if size < 16 || size%4 != 0 {
		return nil // invalid ftyp box
	}

	box := mpeg4FtpyBox{
		majorBrand:   string(header[8:12]),
		minorVersion: binary.BigEndian.Uint32(header[12:16]),
	}

	// compatible brands
	for i := 16; i < int(size) && i+4 < len(header); i += 4 {
		box.compatibleBrands = append(box.compatibleBrands, string(header[i:i+4]))
	}

	return &box
}

// mp3Sniffer detects MP3 files with or without ID3v2 tags
type mp3Sniffer struct{}

func (m *mp3Sniffer) Sniff(header []byte) bool {
	if len(header) < 4 {
		return false
	}

	// Check for ID3v2 tag first (most common)
	if bytes.HasPrefix(header, []byte("ID3")) {
		return true
	}

	// Check for MP3 frame header (for files without ID3v2 tag)
	return m.isMP3FrameHeader(header)
}

// isMP3FrameHeader checks if the header contains a valid MP3 frame sync
func (m *mp3Sniffer) isMP3FrameHeader(header []byte) bool {
	// MP3 frame header starts with 11 bits of sync (all 1s): 0xFFE0 or higher
	// We need at least 4 bytes to check the frame header
	for i := 0; i <= len(header)-4; i++ {
		if m.isValidMP3Frame(header[i:]) {
			return true
		}
	}
	return false
}

// isValidMP3Frame checks if 4 bytes represent a valid MP3 frame header
func (m *mp3Sniffer) isValidMP3Frame(frame []byte) bool {
	if len(frame) < 4 {
		return false
	}

	// Check sync bits (first 11 bits should be all 1s)
	if frame[0] != 0xFF || (frame[1]&0xE0) != 0xE0 {
		return false
	}

	// Check MPEG version (bits 19-20)
	version := (frame[1] >> 3) & 0x03
	if version == 1 { // reserved version
		return false
	}

	// Check layer (bits 17-18)
	layer := (frame[1] >> 1) & 0x03
	if layer == 0 { // reserved layer
		return false
	}

	// Check bitrate (bits 12-15)
	bitrate := (frame[2] >> 4) & 0x0F
	if bitrate == 0 || bitrate == 15 { // free or reserved bitrate
		return false
	}

	// Check sampling frequency (bits 10-11)
	samplingFreq := (frame[2] >> 2) & 0x03
	if samplingFreq == 3 { // reserved sampling frequency
		return false
	}

	return true
}
