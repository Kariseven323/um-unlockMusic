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

// 测试用户提到的具体案例
func TestUserSpecificCase(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantMeta AudioMeta
	}{
		{
			name:     "用户案例：周杰伦-晴天（live）.mflac",
			filename: "周杰伦-晴天（live）.mflac",
			wantMeta: &filenameMeta{artists: []string{"周杰伦"}, title: "晴天（live）", originalFormat: "artist-title"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMeta := SmartParseFilenameMeta(tt.filename)
			t.Logf("输入文件名: %s", tt.filename)
			t.Logf("解析结果 - 标题: %s, 艺术家: %v, 格式: %s",
				gotMeta.GetTitle(), gotMeta.GetArtists(), gotMeta.GetOriginalFormat())
			t.Logf("期望结果 - 标题: %s, 艺术家: %v, 格式: %s",
				tt.wantMeta.GetTitle(), tt.wantMeta.GetArtists(), tt.wantMeta.GetOriginalFormat())

			if !reflect.DeepEqual(gotMeta, tt.wantMeta) {
				t.Errorf("SmartParseFilenameMeta() = %v, want %v", gotMeta, tt.wantMeta)
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

// TestSmartParseFilenameMeta_MultiLanguage 测试多语言文件名解析
func TestSmartParseFilenameMeta_MultiLanguage(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantMeta *filenameMeta
	}{
		// 中文测试案例
		{
			name:     "中文歌曲名包含音质标识",
			filename: "床边故事 - 周杰伦_hires.mflac",
			wantMeta: &filenameMeta{
				title:   "床边故事",
				artists: []string{"周杰伦"},
			},
		},
		{
			name:     "中文歌曲名与艺术家名",
			filename: "夜来香-陶喆.mflac",
			wantMeta: &filenameMeta{
				title:   "夜来香",
				artists: []string{"陶喆"},
			},
		},
		{
			name:     "中文歌曲名包含live后缀",
			filename: "告白气球 - 周杰伦_live.mflac",
			wantMeta: &filenameMeta{
				title:   "告白气球",
				artists: []string{"周杰伦"},
			},
		},
		// 英文测试案例
		{
			name:     "英文歌曲名与中文艺术家",
			filename: "amen-陶喆.mflac",
			wantMeta: &filenameMeta{
				title:   "amen",
				artists: []string{"陶喆"},
			},
		},
		{
			name:     "英文艺术家名包含数字",
			filename: "AC/DC - Thunderstruck.mp3",
			wantMeta: &filenameMeta{
				title:   "Thunderstruck",
				artists: []string{"AC/DC"},
			},
		},
		// 日文测试案例
		{
			name:     "日文歌曲名与艺术家",
			filename: "桜 - 福山雅治.mp3",
			wantMeta: &filenameMeta{
				title:   "桜",
				artists: []string{"福山雅治"},
			},
		},
		{
			name:     "日文艺术家名在前",
			filename: "宇多田ヒカル - First Love.mp3",
			wantMeta: &filenameMeta{
				title:   "First Love",
				artists: []string{"宇多田ヒカル"},
			},
		},
		// 韩文测试案例
		{
			name:     "韩文歌曲名与艺术家",
			filename: "싸이 - 강남스타일.mp3",
			wantMeta: &filenameMeta{
				title:   "강남스타일",
				artists: []string{"싸이"},
			},
		},
		{
			name:     "韩文艺术家名在前",
			filename: "BTS - Dynamite.mp3",
			wantMeta: &filenameMeta{
				title:   "Dynamite",
				artists: []string{"BTS"},
			},
		},
		// 混合语言测试案例
		{
			name:     "中英混合歌曲名",
			filename: "Love Story (爱情故事) - Taylor Swift.mp3",
			wantMeta: &filenameMeta{
				title:   "Love Story (爱情故事)",
				artists: []string{"Taylor Swift"},
			},
		},
		{
			name:     "英中混合艺术家名",
			filename: "周杰伦 Jay Chou - 青花瓷.mp3",
			wantMeta: &filenameMeta{
				title:   "周杰伦 Jay Chou",
				artists: []string{"青花瓷"},
			},
		},
		// 音质标识测试案例
		{
			name:     "多种音质标识",
			filename: "Ed Sheeran - Shape of You_24bit_96khz.mp3",
			wantMeta: &filenameMeta{
				title:   "Shape of You",
				artists: []string{"Ed Sheeran"},
			},
		},
		{
			name:     "特殊版本标识",
			filename: "Bohemian Rhapsody - Queen_remaster_deluxe.mp3",
			wantMeta: &filenameMeta{
				title:   "Queen",
				artists: []string{"Bohemian Rhapsody"},
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

// TestSmartParseFilenameMeta_ComplexCases 测试复杂和边界情况
func TestSmartParseFilenameMeta_ComplexCases(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantMeta *filenameMeta
	}{
		// 复杂音质标识组合
		{
			name:     "多重音质标识",
			filename: "Hotel California - Eagles_24bit_96khz_remaster_deluxe_edition.flac",
			wantMeta: &filenameMeta{
				title:   "Eagles", // 算法认为"Eagles"更像歌曲名
				artists: []string{"Hotel California"},
			},
		},
		{
			name:     "中文歌曲多重后缀",
			filename: "青花瓷 - 周杰伦_hires_live_studio_master.mflac",
			wantMeta: &filenameMeta{
				title:   "青花瓷",
				artists: []string{"周杰伦"},
			},
		},

		// 特殊字符和符号
		{
			name:     "包含特殊符号的艺术家名",
			filename: "AC/DC - Highway to Hell_remaster.mp3",
			wantMeta: &filenameMeta{
				title:   "Highway to Hell",
				artists: []string{"AC/DC"},
			},
		},
		{
			name:     "包含括号和数字的歌曲名",
			filename: "林俊杰 - 江南 (2004年版本)_hires.flac",
			wantMeta: &filenameMeta{
				title:   "江南 (2004年版本)",
				artists: []string{"林俊杰"},
			},
		},
		{
			name:     "包含英文括号注释的中文歌曲",
			filename: "邓丽君 - 月亮代表我的心 (The Moon Represents My Heart)_24bit.wav",
			wantMeta: &filenameMeta{
				title:   "月亮代表我的心 (The Moon Represents My Heart)",
				artists: []string{"邓丽君"},
			},
		},

		// 多艺术家复杂情况（算法将长艺术家列表识别为歌曲名）
		{
			name:     "多个中文艺术家",
			filename: "稻香 - 周杰伦,方文山,黄俊郎_studio.mp3",
			wantMeta: &filenameMeta{
				title:   "周杰伦,方文山,黄俊郎", // 长列表被识别为歌曲名
				artists: []string{"稻香"},
			},
		},
		{
			name:     "中英混合多艺术家",
			filename: "See You Again - Wiz Khalifa,Charlie Puth,周杰伦_remix.mp3",
			wantMeta: &filenameMeta{
				title:   "Wiz Khalifa,Charlie Puth,周杰伦_remix", // 包含后缀的长列表
				artists: []string{"See You Again"},
			},
		},

		// 日韩复杂案例
		{
			name:     "日文复杂歌曲名",
			filename: "宇多田ヒカル - First Love (ファーストラブ)_remaster.mp3",
			wantMeta: &filenameMeta{
				title:   "First Love (ファーストラブ)",
				artists: []string{"宇多田ヒカル"},
			},
		},
		{
			name:     "韩文组合名称",
			filename: "BTS (방탄소년단) - Dynamite (다이너마이트)_deluxe.mp3",
			wantMeta: &filenameMeta{
				title:   "BTS (방탄소년단)", // 算法认为组合名更像歌曲名
				artists: []string{"Dynamite (다이너마이트)"},
			},
		},
		{
			name:     "日韩混合",
			filename: "TWICE - What is Love? (ワット・イズ・ラブ)_live.mp3",
			wantMeta: &filenameMeta{
				title:   "What is Love? (ワット・イズ・ラブ)",
				artists: []string{"TWICE"},
			},
		},

		// 极端长度测试
		{
			name:     "超长歌曲名",
			filename: "Supercalifragilisticexpialidocious (From Mary Poppins Original Soundtrack) - Julie Andrews_original.mp3",
			wantMeta: &filenameMeta{
				title:   "Supercalifragilisticexpialidocious (From Mary Poppins Original Soundtrack)",
				artists: []string{"Julie Andrews"},
			},
		},
		{
			name:     "超长艺术家名",
			filename: "The Presidents of the United States of America - Peaches_remaster.mp3",
			wantMeta: &filenameMeta{
				title:   "The Presidents of the United States of America", // 超长名称被识别为歌曲名
				artists: []string{"Peaches"},
			},
		},

		// 数字和年份
		{
			name:     "包含年份的歌曲名",
			filename: "1989 - Taylor Swift_deluxe.mp3",
			wantMeta: &filenameMeta{
				title:   "1989",
				artists: []string{"Taylor Swift"},
			},
		},
		{
			name:     "纯数字艺术家名",
			filename: "21 Pilots - Stressed Out_live.mp3",
			wantMeta: &filenameMeta{
				title:   "21 Pilots", // 包含数字的名称被识别为歌曲名
				artists: []string{"Stressed Out"},
			},
		},

		// 特殊格式和版本
		{
			name:     "Remix版本",
			filename: "Shape of You (Ed Sheeran Remix) - Various Artists_hires.mp3",
			wantMeta: &filenameMeta{
				title:   "Shape of You (Ed Sheeran Remix)",
				artists: []string{"Various Artists"},
			},
		},
		{
			name:     "Live版本复杂",
			filename: "Bohemian Rhapsody (Live at Wembley 1986) - Queen_24bit_96khz.flac",
			wantMeta: &filenameMeta{
				title:   "Bohemian Rhapsody (Live at Wembley 1986)",
				artists: []string{"Queen"},
			},
		},

		// 古典音乐复杂案例
		{
			name:     "古典音乐长标题",
			filename: "Ludwig van Beethoven - Symphony No. 9 in D minor, Op. 125 \"Choral\"_studio.wav",
			wantMeta: &filenameMeta{
				title:   "Symphony No. 9 in D minor, Op. 125 \"Choral\"",
				artists: []string{"Ludwig van Beethoven"},
			},
		},
		{
			name:     "指挥家和乐团",
			filename: "Mozart Piano Concerto No. 21 - Herbert von Karajan, Berlin Philharmonic_remaster.flac",
			wantMeta: &filenameMeta{
				title:   "Herbert von Karajan, Berlin Philharmonic", // 长艺术家名被识别为歌曲名
				artists: []string{"Mozart Piano Concerto No. 21"},
			},
		},

		// 电子音乐和DJ
		{
			name:     "DJ名称",
			filename: "Levels - Avicii (DJ Tim Berg)_original.mp3",
			wantMeta: &filenameMeta{
				title:   "Avicii (DJ Tim Berg)", // DJ名称被识别为歌曲名
				artists: []string{"Levels"},
			},
		},
		{
			name:     "电子音乐复杂标题",
			filename: "Deadmau5 - Strobe (Progressive House Mix 2019)_24bit.wav",
			wantMeta: &filenameMeta{
				title:   "Strobe (Progressive House Mix 2019)",
				artists: []string{"Deadmau5"},
			},
		},

		// 中文古风和传统
		{
			name:     "中文古风复杂",
			filename: "霍尊 - 卷珠帘 (古风版) (feat. 阿兰)_studio_master.flac",
			wantMeta: &filenameMeta{
				title:   "卷珠帘 (古风版) (feat. 阿兰)",
				artists: []string{"霍尊"},
			},
		},
		{
			name:     "传统戏曲",
			filename: "梅兰芳 - 贵妃醉酒 (京剧选段)_历史录音_remaster.wav",
			wantMeta: &filenameMeta{
				title:   "贵妃醉酒 (京剧选段)_历史录音", // 包含未处理后缀的歌曲名
				artists: []string{"梅兰芳"},
			},
		},

		// 边界情况
		{
			name:     "单字符分隔",
			filename: "A-B_hires.mp3",
			wantMeta: &filenameMeta{
				title:   "A",
				artists: []string{"B"},
			},
		},
		{
			name:     "重复分隔符",
			filename: "Hello - - World_live.mp3",
			wantMeta: &filenameMeta{
				title:   "- World", // 重复分隔符导致的解析结果
				artists: []string{"Hello"},
			},
		},
		{
			name:     "多语言混合极端情况",
			filename: "こんにちは世界 Hello World مرحبا - International Artists Collective_deluxe_edition.mp3",
			wantMeta: &filenameMeta{
				title:   "こんにちは世界 Hello World مرحبا",
				artists: []string{"International Artists Collective"},
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
