//go:build windows
// +build windows

package mmap

import (
	"fmt"
	"io"
	"os"
)

// MmapReader 内存映射文件读取器（Windows版本 - 使用常规文件读取）
type MmapReader struct {
	file   *os.File
	offset int64
	size   int64
}

// NewMmapReader 创建内存映射读取器（Windows版本）
func NewMmapReader(filename string) (*MmapReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	return &MmapReader{
		file:   file,
		offset: 0,
		size:   stat.Size(),
	}, nil
}

// Read 读取数据
func (r *MmapReader) Read(p []byte) (n int, err error) {
	if r.offset >= r.size {
		return 0, io.EOF
	}

	n, err = r.file.ReadAt(p, r.offset)
	r.offset += int64(n)
	return n, err
}

// Seek 设置读取位置
func (r *MmapReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		r.offset = offset
	case io.SeekCurrent:
		r.offset += offset
	case io.SeekEnd:
		r.offset = r.size + offset
	default:
		return 0, fmt.Errorf("无效的whence值: %d", whence)
	}

	if r.offset < 0 {
		r.offset = 0
	}
	if r.offset > r.size {
		r.offset = r.size
	}

	return r.offset, nil
}

// Close 关闭文件
func (r *MmapReader) Close() error {
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

// Size 返回文件大小
func (r *MmapReader) Size() int64 {
	return r.size
}
