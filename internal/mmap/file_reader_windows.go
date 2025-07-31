//go:build windows
// +build windows

package mmap

import (
	"fmt"
	"io"
	"os"
	"syscall"
	"unsafe"
)

// Windows API 常量
const (
	INVALID_HANDLE_VALUE = ^uintptr(0)
	FILE_MAP_READ        = 0x0004
	PAGE_READONLY        = 0x02
)

// Windows API 函数
var (
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	procCreateFileMapping = kernel32.NewProc("CreateFileMappingW")
	procMapViewOfFile     = kernel32.NewProc("MapViewOfFile")
	procUnmapViewOfFile   = kernel32.NewProc("UnmapViewOfFile")
	procCloseHandle       = kernel32.NewProc("CloseHandle")
)

// MmapReader 内存映射文件读取器（Windows版本 - 真正的内存映射实现）
type MmapReader struct {
	file          *os.File
	data          []byte
	offset        int64
	size          int64
	mappingHandle uintptr
	useMmap       bool
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

	size := stat.Size()
	if size == 0 {
		file.Close()
		return nil, fmt.Errorf("文件为空")
	}

	// 尝试创建内存映射
	data, mappingHandle, err := mmapWindows(file, size)
	if err != nil {
		// 内存映射失败，回退到常规文件读取
		return &MmapReader{
			file:    file,
			offset:  0,
			size:    size,
			useMmap: false,
		}, nil
	}

	return &MmapReader{
		file:          file,
		data:          data,
		offset:        0,
		size:          size,
		mappingHandle: mappingHandle,
		useMmap:       true,
	}, nil
}

// Read 读取数据
func (r *MmapReader) Read(p []byte) (n int, err error) {
	if r.offset >= r.size {
		return 0, io.EOF
	}

	if r.useMmap && r.data != nil {
		// 使用内存映射读取
		available := r.size - r.offset
		if int64(len(p)) > available {
			p = p[:available]
		}

		copy(p, r.data[r.offset:r.offset+int64(len(p))])
		r.offset += int64(len(p))
		return len(p), nil
	} else {
		// 回退到常规文件读取
		n, err = r.file.ReadAt(p, r.offset)
		r.offset += int64(n)
		return n, err
	}
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
	var err error

	// 如果使用内存映射，先取消映射
	if r.useMmap && r.data != nil {
		if unmapErr := munmapWindows(r.data, r.mappingHandle); unmapErr != nil {
			err = unmapErr
		}
		r.data = nil
		r.mappingHandle = 0
	}

	// 关闭文件
	if r.file != nil {
		if closeErr := r.file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		r.file = nil
	}

	return err
}

// Size 返回文件大小
func (r *MmapReader) Size() int64 {
	return r.size
}

// mmapWindows Windows平台的内存映射实现
func mmapWindows(file *os.File, size int64) ([]byte, uintptr, error) {
	// 添加大小限制检查
	const maxMmapSize = 1 << 30 // 1GB限制
	if size <= 0 {
		return nil, 0, fmt.Errorf("invalid file size: %d", size)
	}
	if size > maxMmapSize {
		return nil, 0, fmt.Errorf("file too large for memory mapping: %d bytes (max: %d)", size, maxMmapSize)
	}

	// 获取文件句柄
	fileHandle := syscall.Handle(file.Fd())
	if fileHandle == syscall.InvalidHandle {
		return nil, 0, fmt.Errorf("invalid file handle")
	}

	// 创建文件映射对象
	mappingHandle, _, err := procCreateFileMapping.Call(
		uintptr(fileHandle),
		0, // 默认安全属性
		PAGE_READONLY,
		0,             // 高32位大小
		uintptr(size), // 低32位大小
		0,             // 映射名称（匿名）
	)

	if mappingHandle == 0 {
		// 获取具体的Windows错误码
		if errno, ok := err.(syscall.Errno); ok {
			return nil, 0, fmt.Errorf("CreateFileMapping failed: %v (error code: %d)", err, errno)
		}
		return nil, 0, fmt.Errorf("CreateFileMapping failed: %v", err)
	}

	// 映射视图到进程地址空间
	viewPtr, _, err := procMapViewOfFile.Call(
		mappingHandle,
		FILE_MAP_READ,
		0, // 高32位偏移
		0, // 低32位偏移
		uintptr(size),
	)

	if viewPtr == 0 {
		procCloseHandle.Call(mappingHandle)
		if errno, ok := err.(syscall.Errno); ok {
			return nil, 0, fmt.Errorf("MapViewOfFile failed: %v (error code: %d)", err, errno)
		}
		return nil, 0, fmt.Errorf("MapViewOfFile failed: %v", err)
	}

	// 安全地将指针转换为字节切片
	if size > 1<<30 {
		procCloseHandle.Call(mappingHandle)
		return nil, 0, fmt.Errorf("size too large for slice conversion: %d", size)
	}
	data := unsafe.Slice((*byte)(unsafe.Pointer(viewPtr)), size)

	return data, mappingHandle, nil
}

// munmapWindows Windows平台的内存映射取消实现
func munmapWindows(data []byte, mappingHandle uintptr) error {
	var err error

	// 取消视图映射
	if len(data) > 0 {
		dataPtr := uintptr(unsafe.Pointer(&data[0]))
		ret, _, winErr := procUnmapViewOfFile.Call(dataPtr)
		if ret == 0 {
			if errno, ok := winErr.(syscall.Errno); ok {
				err = fmt.Errorf("UnmapViewOfFile failed: %v (error code: %d)", winErr, errno)
			} else {
				err = fmt.Errorf("UnmapViewOfFile failed: %v", winErr)
			}
		}
	}

	// 关闭映射句柄
	if mappingHandle != 0 {
		ret, _, winErr := procCloseHandle.Call(mappingHandle)
		if ret == 0 && err == nil {
			if errno, ok := winErr.(syscall.Errno); ok {
				err = fmt.Errorf("CloseHandle failed: %v (error code: %d)", winErr, errno)
			} else {
				err = fmt.Errorf("CloseHandle failed: %v", winErr)
			}
		}
	}

	return err
}
