package tm

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"unlock-music.dev/cli/algo/common"
	"unlock-music.dev/cli/internal/mmap"
	"unlock-music.dev/cli/internal/pool"
	"unlock-music.dev/cli/internal/sniff"
)

var replaceHeader = []byte{0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70}
var magicHeader = []byte{0x51, 0x51, 0x4D, 0x55} //0x15, 0x1D, 0x1A, 0x21

type Decoder struct {
	raw io.ReadSeeker // raw is the original file reader

	offset int
	audio  io.Reader // audio is the decrypted audio data

	// 优化相关字段
	fileSize     int64
	useOptimized bool
	mmapReader   *mmap.MmapReader
}

func (d *Decoder) Validate() error {
	// 尝试获取文件大小以优化处理
	if file, ok := d.raw.(*os.File); ok {
		if stat, err := file.Stat(); err == nil {
			d.fileSize = stat.Size()
		}
	}

	// 使用内存池获取header缓冲区
	headerBuf := pool.GetSmallBuffer()
	defer pool.PutBuffer(headerBuf)

	header := headerBuf[:8] // 只使用前8字节
	if _, err := io.ReadFull(d.raw, header); err != nil {
		return fmt.Errorf("tm read header: %w", err)
	}

	if bytes.Equal(magicHeader, header[:len(magicHeader)]) { // replace m4a header
		d.audio = d.createOptimizedReader(replaceHeader)
		return nil
	}

	if _, ok := sniff.AudioExtension(header); ok { // not encrypted
		d.audio = d.createOptimizedReader(header)
		return nil
	}

	return errors.New("tm: valid magic header")
}

func (d *Decoder) Read(buf []byte) (int, error) {
	return d.audio.Read(buf)
}

func NewTmDecoder(p *common.DecoderParams) common.Decoder {
	return &Decoder{raw: p.Reader}
}

// createOptimizedReader 创建优化的读取器
func (d *Decoder) createOptimizedReader(headerReplacement []byte) io.Reader {
	// 对于大文件，尝试使用内存映射
	if d.fileSize > 10*1024*1024 { // >10MB
		if file, ok := d.raw.(*os.File); ok {
			if mmapReader, err := mmap.NewMmapReader(file.Name()); err == nil {
				d.mmapReader = mmapReader
				d.useOptimized = true

				// 跳过已读取的header部分
				if _, err := mmapReader.Seek(8, io.SeekStart); err == nil {
					return io.MultiReader(bytes.NewReader(headerReplacement), mmapReader)
				}
				// 如果seek失败，关闭mmap并回退
				mmapReader.Close()
			}
		}
	}

	// 回退到标准MultiReader
	return io.MultiReader(bytes.NewReader(headerReplacement), d.raw)
}

// Close 关闭解码器，清理资源
func (d *Decoder) Close() error {
	if d.mmapReader != nil {
		return d.mmapReader.Close()
	}
	return nil
}

func init() {
	// QQ Music IOS M4a (replace header)
	common.RegisterDecoder("tm2", false, NewTmDecoder)
	common.RegisterDecoder("tm6", false, NewTmDecoder)

	// QQ Music IOS Mp3 (not encrypted)
	common.RegisterDecoder("tm0", false, NewTmDecoder)
	common.RegisterDecoder("tm3", false, NewTmDecoder)
}
