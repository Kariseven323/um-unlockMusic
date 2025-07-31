package main

import (
	"encoding/json"
	"testing"

	"unlock-music.dev/cli/algo/common"
)

// TestBatchSingleFileConsistency 测试batch模式和单文件模式的一致性
func TestBatchSingleFileConsistency(t *testing.T) {
	// 测试用例：不同的文件名格式
	testCases := []struct {
		name     string
		filename string
		expected struct {
			title   string
			artists []string
		}
	}{
		{
			name:     "艺术家-标题格式",
			filename: "周杰伦 - 晴天.ncm",
			expected: struct {
				title   string
				artists []string
			}{
				title:   "晴天",
				artists: []string{"周杰伦"},
			},
		},
		{
			name:     "标题-艺术家格式",
			filename: "晴天 - 周杰伦.ncm",
			expected: struct {
				title   string
				artists []string
			}{
				title:   "晴天",
				artists: []string{"周杰伦"},
			},
		},
		{
			name:     "英文艺术家-标题格式",
			filename: "Taylor Swift - Love Story.ncm",
			expected: struct {
				title   string
				artists []string
			}{
				title:   "Love Story",
				artists: []string{"Taylor Swift"},
			},
		},
		{
			name:     "包含Live后缀的标题",
			filename: "听妈妈的话 (Live) - 周杰伦.ncm",
			expected: struct {
				title   string
				artists []string
			}{
				title:   "听妈妈的话 (Live)",
				artists: []string{"周杰伦"},
			},
		},
		{
			name:     "简单艺术家名",
			filename: "青花瓷 - 周杰伦.ncm",
			expected: struct {
				title   string
				artists []string
			}{
				title:   "青花瓷",
				artists: []string{"周杰伦"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 测试智能解析函数的一致性
			meta := common.SmartParseFilenameMeta(tc.filename)

			// 验证标题
			if meta.GetTitle() != tc.expected.title {
				t.Errorf("标题解析不一致: 期望 %q, 实际 %q", tc.expected.title, meta.GetTitle())
			}

			// 验证艺术家
			artists := meta.GetArtists()
			if len(artists) != len(tc.expected.artists) {
				t.Errorf("艺术家数量不一致: 期望 %d, 实际 %d", len(tc.expected.artists), len(artists))
			} else {
				for i, expectedArtist := range tc.expected.artists {
					if i < len(artists) && artists[i] != expectedArtist {
						t.Errorf("艺术家[%d]不一致: 期望 %q, 实际 %q", i, expectedArtist, artists[i])
					}
				}
			}
		})
	}
}

// TestNamingFormatConsistency 测试不同命名格式的一致性
func TestNamingFormatConsistency(t *testing.T) {
	testCases := []struct {
		name         string
		filename     string
		namingFormat string
		expectedName string
	}{
		{
			name:         "auto格式-艺术家标题",
			filename:     "周杰伦 - 晴天",
			namingFormat: "auto",
			expectedName: "周杰伦 - 晴天.mp3",
		},
		{
			name:         "title-artist格式",
			filename:     "周杰伦 - 晴天",
			namingFormat: "title-artist",
			expectedName: "晴天 - 周杰伦.mp3",
		},
		{
			name:         "artist-title格式",
			filename:     "周杰伦 - 晴天",
			namingFormat: "artist-title",
			expectedName: "周杰伦 - 晴天.mp3",
		},
		{
			name:         "original格式",
			filename:     "周杰伦 - 晴天",
			namingFormat: "original",
			expectedName: "周杰伦 - 晴天.mp3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建模拟的processor
			p := &processor{
				namingFormat: tc.namingFormat,
			}

			// 测试文件名生成
			result := p.generateOutputFilename(tc.filename, ".mp3")

			if result != tc.expectedName {
				t.Errorf("文件名生成不一致: 期望 %q, 实际 %q", tc.expectedName, result)
			}
		})
	}
}

// TestMetaWrapperConsistency 测试元数据包装器的一致性
func TestMetaWrapperConsistency(t *testing.T) {
	// 模拟原始元数据（可能包含错误的标题）
	originalMeta := &mockAudioMeta{
		title:   "错误的标题",
		album:   "正确的专辑",
		artists: []string{"正确的艺术家"},
	}

	filename := "正确的艺术家 - 正确的标题.ncm"

	// 使用WrapMetaWithFilename包装
	wrappedMeta := common.WrapMetaWithFilename(originalMeta, filename)

	// 验证标题优先使用文件名中的信息
	if wrappedMeta.GetTitle() != "正确的标题" {
		t.Errorf("包装后的标题不正确: 期望 %q, 实际 %q", "正确的标题", wrappedMeta.GetTitle())
	}

	// 验证艺术家信息优先使用原始元数据
	artists := wrappedMeta.GetArtists()
	if len(artists) != 1 || artists[0] != "正确的艺术家" {
		t.Errorf("包装后的艺术家信息不正确: 期望 [%q], 实际 %v", "正确的艺术家", artists)
	}

	// 验证专辑信息优先使用原始元数据
	if wrappedMeta.GetAlbum() != "正确的专辑" {
		t.Errorf("包装后的专辑信息不正确: 期望 %q, 实际 %q", "正确的专辑", wrappedMeta.GetAlbum())
	}
}

// mockAudioMeta 模拟音频元数据
type mockAudioMeta struct {
	title   string
	album   string
	artists []string
}

func (m *mockAudioMeta) GetTitle() string {
	return m.title
}

func (m *mockAudioMeta) GetAlbum() string {
	return m.album
}

func (m *mockAudioMeta) GetArtists() []string {
	return m.artists
}

func (m *mockAudioMeta) GetOriginalFormat() string {
	return ""
}

// TestBatchRequestProcessing 测试批处理请求的处理
func TestBatchRequestProcessing(t *testing.T) {
	// 创建测试批处理请求
	request := &BatchRequest{
		Files: []FileTask{
			{InputPath: "test1.ncm"},
			{InputPath: "test2.qmc"},
		},
		Options: ProcessOptions{
			NamingFormat:    "auto",
			UpdateMetadata:  true,
			OverwriteOutput: true,
			RemoveSource:    false,
			SkipNoop:        true,
		},
	}

	// 验证默认命名格式设置
	if request.Options.NamingFormat != "auto" {
		t.Errorf("默认命名格式不正确: 期望 %q, 实际 %q", "auto", request.Options.NamingFormat)
	}

	// 测试JSON序列化和反序列化
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("JSON序列化失败: %v", err)
	}

	var decoded BatchRequest
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("JSON反序列化失败: %v", err)
	}

	// 验证反序列化后的数据
	if len(decoded.Files) != 2 {
		t.Errorf("文件数量不一致: 期望 2, 实际 %d", len(decoded.Files))
	}

	if decoded.Options.NamingFormat != "auto" {
		t.Errorf("命名格式不一致: 期望 %q, 实际 %q", "auto", decoded.Options.NamingFormat)
	}
}
