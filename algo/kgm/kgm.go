package kgm

import (
	"fmt"
	"io"

	"unlock-music.dev/cli/algo/common"
)

type Decoder struct {
	rd io.ReadSeeker

	cipher common.StreamDecoder
	offset int

	header header

	KggDatabasePath string
}

func NewDecoder(p *common.DecoderParams) common.Decoder {
	return &Decoder{rd: p.Reader, KggDatabasePath: p.KggDatabasePath}
}

// Validate checks if the file is a valid Kugou (.kgm, .vpr, .kgma) file.
// rd will be seeked to the beginning of the encrypted audio.
func (d *Decoder) Validate() (err error) {
	if err := d.header.FromFile(d.rd); err != nil {
		return err
	}
	// Validate crypto version and initialize cipher
	switch d.header.CryptoVersion {
	case 3:
		d.cipher, err = newKgmCryptoV3(&d.header)
		if err != nil {
			return fmt.Errorf("kgm init crypto v3: %w", err)
		}
	case 5:
		d.cipher, err = newKgmCryptoV5(&d.header, d.KggDatabasePath)
		if err != nil {
			return fmt.Errorf("kgm init crypto v5: %w", err)
		}
	case 1, 2:
		return fmt.Errorf("kgm: crypto version %d is deprecated and no longer supported", d.header.CryptoVersion)
	case 4:
		return fmt.Errorf("kgm: crypto version %d was experimental and is not supported", d.header.CryptoVersion)
	default:
		if d.header.CryptoVersion > 5 {
			return fmt.Errorf("kgm: crypto version %d is newer than supported (max: 5). Please update the decoder", d.header.CryptoVersion)
		}
		return fmt.Errorf("kgm: invalid crypto version %d", d.header.CryptoVersion)
	}

	// prepare for read
	if _, err := d.rd.Seek(int64(d.header.AudioOffset), io.SeekStart); err != nil {
		return fmt.Errorf("kgm seek to audio: %w", err)
	}

	return nil
}

func (d *Decoder) Read(buf []byte) (int, error) {
	n, err := d.rd.Read(buf)
	if n > 0 {
		d.cipher.Decrypt(buf[:n], d.offset)
		d.offset += n
	}
	return n, err
}

func init() {
	// Kugou
	common.RegisterDecoder("kgg", false, NewDecoder)
	common.RegisterDecoder("kgm", false, NewDecoder)
	common.RegisterDecoder("kgma", false, NewDecoder)
	// Viper
	common.RegisterDecoder("vpr", false, NewDecoder)
	// Kugou Android
	common.RegisterDecoder("kgm.flac", false, NewDecoder)
	common.RegisterDecoder("vpr.flac", false, NewDecoder)
}
