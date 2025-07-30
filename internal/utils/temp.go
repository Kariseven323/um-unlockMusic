package utils

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
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
// 支持格式: "标题 - 艺术家.ext" 或 "标题 (Live) - 艺术家.ext"
func ExtractTitleFromFilename(filename string) string {
	// 移除文件扩展名
	name := filename
	if lastDot := strings.LastIndex(name, "."); lastDot != -1 {
		name = name[:lastDot]
	}

	// 使用正则表达式匹配 "标题 - 艺术家" 格式
	// 匹配到第一个 " - " 之前的部分作为标题
	re := regexp.MustCompile(`^(.+?)\s+-\s+.+$`)
	matches := re.FindStringSubmatch(name)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}

	// 如果没有匹配到，返回整个文件名（去除扩展名）
	return strings.TrimSpace(name)
}
