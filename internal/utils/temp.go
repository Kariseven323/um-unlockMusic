package utils

import (
	"fmt"
	"io"
	"os"
	"strings"

	"unlock-music.dev/cli/algo/common"
	"unlock-music.dev/cli/internal/pool"
)

func WriteTempFile(rd io.Reader, ext string) (string, error) {
	audioFile, err := os.CreateTemp("", "*"+ext)
	if err != nil {
		return "", fmt.Errorf("ffmpeg create temp file: %w", err)
	}

	if _, err := io.Copy(audioFile, rd); err != nil {
		return "", fmt.Errorf("ffmpeg write temp file: %w", err)
	}

	if err := audioFile.Close(); err != nil {
		return "", fmt.Errorf("ffmpeg close temp file: %w", err)
	}

	return audioFile.Name(), nil
}

// ExtractTitleFromFilename 从文件名中提取标题
// 使用智能解析算法，支持多种命名格式：
// 1. "艺术家 - 标题.ext" (如: "周杰伦 - 晴天.flac")
// 2. "标题 - 艺术家.ext" (如: "晴天 - 周杰伦.flac")
func ExtractTitleFromFilename(filename string) string {
	// 使用智能解析获取元数据
	meta := common.SmartParseFilenameMeta(filename)

	// 返回解析出的标题
	if title := meta.GetTitle(); title != "" {
		return title
	}

	// 简化的回退逻辑：直接返回去除扩展名的文件名
	name := filename
	if lastDot := strings.LastIndex(name, "."); lastDot != -1 {
		name = name[:lastDot]
	}

	return strings.TrimSpace(name)
}

// OptimizedCopy 使用内存池优化的文件复制函数
func OptimizedCopy(dst io.Writer, src io.Reader) (written int64, err error) {
	// 使用4MB缓冲区进行高效I/O
	buf := pool.GetXLargeBuffer()
	defer pool.PutBuffer(buf)

	return io.CopyBuffer(dst, src, buf)
}

// OptimizedWriteTempFile 使用优化I/O的临时文件写入
func OptimizedWriteTempFile(rd io.Reader, ext string) (string, error) {
	audioFile, err := os.CreateTemp("", "*"+ext)
	if err != nil {
		return "", fmt.Errorf("ffmpeg create temp file: %w", err)
	}

	if _, err := OptimizedCopy(audioFile, rd); err != nil {
		return "", fmt.Errorf("ffmpeg write temp file: %w", err)
	}

	if err := audioFile.Close(); err != nil {
		return "", fmt.Errorf("ffmpeg close temp file: %w", err)
	}

	return audioFile.Name(), nil
}
