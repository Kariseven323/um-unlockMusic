package common

import (
	"reflect"
	"testing"
)

func TestParseFilenameMeta(t *testing.T) {

	tests := []struct {
		name     string
		wantMeta AudioMeta
	}{
		{
			name:     "test1",
			wantMeta: &filenameMeta{title: "test1", originalFormat: ""},
		},
		{
			name:     "晴天 - 周杰伦.flac",
			wantMeta: &filenameMeta{artists: []string{"周杰伦"}, title: "晴天", originalFormat: ""},
		},
		{
			name:     "Sing Me to Sleep - Alan Walker _ Iselin Solheim.flac",
			wantMeta: &filenameMeta{artists: []string{"Alan Walker", "Iselin Solheim"}, title: "Sing Me to Sleep", originalFormat: ""},
		},
		{
			name:     "Limousine - Christopher,Madcon.flac",
			wantMeta: &filenameMeta{artists: []string{"Christopher", "Madcon"}, title: "Limousine", originalFormat: ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotMeta := ParseFilenameMeta(tt.name); !reflect.DeepEqual(gotMeta, tt.wantMeta) {
				t.Errorf("ParseFilenameMeta() = %v, want %v", gotMeta, tt.wantMeta)
			}
		})
	}
}

func TestSmartParseFilenameMeta(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantMeta AudioMeta
	}{
		{
			name:     "单个词",
			filename: "test1",
			wantMeta: &filenameMeta{title: "test1", originalFormat: "title-only"},
		},
		{
			name:     "艺术家-标题格式（中文）",
			filename: "周杰伦 - 晴天.flac",
			wantMeta: &filenameMeta{artists: []string{"周杰伦"}, title: "晴天", originalFormat: "artist-title"},
		},
		{
			name:     "标题-艺术家格式（中文）",
			filename: "晴天 - 周杰伦.mflac",
			wantMeta: &filenameMeta{artists: []string{"周杰伦"}, title: "晴天", originalFormat: "title-artist"},
		},
		{
			name:     "艺术家-标题格式（英文）",
			filename: "Taylor Swift - Love Story.mp3",
			wantMeta: &filenameMeta{artists: []string{"Taylor Swift"}, title: "Love Story", originalFormat: "artist-title"},
		},
		{
			name:     "标题-艺术家格式（英文）",
			filename: "Love Story - Taylor Swift.mp3",
			wantMeta: &filenameMeta{artists: []string{"Taylor Swift"}, title: "Love Story", originalFormat: "title-artist"},
		},
		{
			name:     "包含Live的歌曲名",
			filename: "听妈妈的话 (Live) - 周杰伦.flac",
			wantMeta: &filenameMeta{artists: []string{"周杰伦"}, title: "听妈妈的话 (Live)", originalFormat: "title-artist"},
		},
		{
			name:     "多个艺术家",
			filename: "Alan Walker,Iselin Solheim - Sing Me to Sleep.flac",
			wantMeta: &filenameMeta{artists: []string{"Alan Walker", "Iselin Solheim"}, title: "Sing Me to Sleep", originalFormat: "artist-title"},
		},
		{
			name:     "复杂歌曲名",
			filename: "Shape of You (Ed Sheeran Cover) - 某艺术家.mp3",
			wantMeta: &filenameMeta{artists: []string{"某艺术家"}, title: "Shape of You (Ed Sheeran Cover)", originalFormat: "title-artist"},
		},
		{
			name:     "空文件名",
			filename: "",
			wantMeta: &filenameMeta{originalFormat: ""},
		},
		{
			name:     "只有扩展名",
			filename: ".mp3",
			wantMeta: &filenameMeta{originalFormat: ""},
		},
		{
			name:     "无分隔符",
			filename: "单个标题.mp3",
			wantMeta: &filenameMeta{title: "单个标题", originalFormat: "title-only"},
		},
		{
			name:     "空的第一部分",
			filename: " - 周杰伦.mp3",
			wantMeta: &filenameMeta{title: "周杰伦", originalFormat: "title-only"},
		},
		{
			name:     "空的第二部分",
			filename: "晴天 - .mp3",
			wantMeta: &filenameMeta{title: "晴天", originalFormat: "title-only"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMeta := SmartParseFilenameMeta(tt.filename)
			if !reflect.DeepEqual(gotMeta, tt.wantMeta) {
				t.Errorf("SmartParseFilenameMeta() = %v, want %v", gotMeta, tt.wantMeta)
				t.Logf("Got title: %s, artists: %v", gotMeta.GetTitle(), gotMeta.GetArtists())
				t.Logf("Want title: %s, artists: %v", tt.wantMeta.GetTitle(), tt.wantMeta.GetArtists())
			}
		})
	}
}

// TestSmartParseFilenameMeta_EdgeCases 测试启发式算法的边界情况
func TestSmartParseFilenameMeta_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantMeta *filenameMeta
	}{
		{
			name:     "包含数字的歌曲名",
			filename: "周杰伦 - 听妈妈的话 (Live).mp3",
			wantMeta: &filenameMeta{
				title:   "听妈妈的话 (Live)",
				artists: []string{"周杰伦"},
			},
		},
		{
			name:     "英文歌曲名包含特殊词汇",
			filename: "Taylor Swift - Love Story (Remix).mp3",
			wantMeta: &filenameMeta{
				title:   "Love Story (Remix)",
				artists: []string{"Taylor Swift"},
			},
		},
		{
			name:     "中英文混合",
			filename: "周杰伦 - Blue and White Porcelain.mp3",
			wantMeta: &filenameMeta{
				title:   "Blue and White Porcelain",
				artists: []string{"周杰伦"},
			},
		},
		{
			name:     "包含特殊字符",
			filename: "周杰伦 - 听妈妈的话 (Live@演唱会).mp3",
			wantMeta: &filenameMeta{
				title:   "听妈妈的话 (Live@演唱会)",
				artists: []string{"周杰伦"},
			},
		},
		{
			name:     "长艺术家名",
			filename: "The Beatles - Yesterday.mp3",
			wantMeta: &filenameMeta{
				title:   "Yesterday",
				artists: []string{"The Beatles"},
			},
		},
		{
			name:     "短艺术家名",
			filename: "U2 - One.mp3",
			wantMeta: &filenameMeta{
				title:   "One",
				artists: []string{"U2"},
			},
		},
		{
			name:     "包含版本信息的标题",
			filename: "周杰伦 - 晴天 (钢琴版).mp3",
			wantMeta: &filenameMeta{
				title:   "晴天 (钢琴版)",
				artists: []string{"周杰伦"},
			},
		},
		{
			name:     "反向格式-标题在前",
			filename: "Love Story - Taylor Swift.mp3",
			wantMeta: &filenameMeta{
				title:   "Love Story",
				artists: []string{"Taylor Swift"},
			},
		},
		{
			name:     "复杂标题包含连字符",
			filename: "周杰伦 - 听妈妈的话-钢琴版.mp3",
			wantMeta: &filenameMeta{
				title:   "听妈妈的话-钢琴版",
				artists: []string{"周杰伦"},
			},
		},
		{
			name:     "空格较多的情况",
			filename: "  周杰伦   -   晴天  .mp3",
			wantMeta: &filenameMeta{
				title:   "晴天",
				artists: []string{"周杰伦"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMeta := SmartParseFilenameMeta(tt.filename)

			// 验证标题
			if gotMeta.GetTitle() != tt.wantMeta.GetTitle() {
				t.Errorf("标题不匹配: got %q, want %q", gotMeta.GetTitle(), tt.wantMeta.GetTitle())
			}

			// 验证艺术家
			gotArtists := gotMeta.GetArtists()
			wantArtists := tt.wantMeta.GetArtists()
			if len(gotArtists) != len(wantArtists) {
				t.Errorf("艺术家数量不匹配: got %d, want %d", len(gotArtists), len(wantArtists))
			} else {
				for i, artist := range wantArtists {
					if i < len(gotArtists) && gotArtists[i] != artist {
						t.Errorf("艺术家[%d]不匹配: got %q, want %q", i, gotArtists[i], artist)
					}
				}
			}
		})
	}
}

func TestSmartParseFilenameMetaOriginalFormat(t *testing.T) {
	tests := []struct {
		name           string
		filename       string
		expectedTitle  string
		expectedArtist string
		expectedFormat string
	}{
		{
			name:           "标题-艺术家格式",
			filename:       "晴天 - 周杰伦.mflac",
			expectedTitle:  "晴天",
			expectedArtist: "周杰伦",
			expectedFormat: "title-artist",
		},
		{
			name:           "艺术家-标题格式",
			filename:       "周杰伦 - 晴天.mflac",
			expectedTitle:  "晴天",
			expectedArtist: "周杰伦",
			expectedFormat: "artist-title",
		},
		{
			name:           "英文标题-艺术家格式",
			filename:       "Love Story - Taylor Swift.mflac",
			expectedTitle:  "Love Story",
			expectedArtist: "Taylor Swift",
			expectedFormat: "title-artist",
		},
		{
			name:           "英文艺术家-标题格式",
			filename:       "Taylor Swift - Love Story.mflac",
			expectedTitle:  "Love Story",
			expectedArtist: "Taylor Swift",
			expectedFormat: "artist-title",
		},
		{
			name:           "只有标题",
			filename:       "单独的歌曲名.mflac",
			expectedTitle:  "单独的歌曲名",
			expectedArtist: "",
			expectedFormat: "title-only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := SmartParseFilenameMeta(tt.filename)

			if meta.GetTitle() != tt.expectedTitle {
				t.Errorf("GetTitle() = %v, want %v", meta.GetTitle(), tt.expectedTitle)
			}

			artists := meta.GetArtists()
			if tt.expectedArtist == "" {
				if len(artists) != 0 {
					t.Errorf("GetArtists() = %v, want empty", artists)
				}
			} else {
				if len(artists) != 1 || artists[0] != tt.expectedArtist {
					t.Errorf("GetArtists() = %v, want [%v]", artists, tt.expectedArtist)
				}
			}

			if meta.GetOriginalFormat() != tt.expectedFormat {
				t.Errorf("GetOriginalFormat() = %v, want %v", meta.GetOriginalFormat(), tt.expectedFormat)
			}
		})
	}
}
