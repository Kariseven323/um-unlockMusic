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
			wantMeta: &filenameMeta{title: "test1"},
		},
		{
			name:     "周杰伦 - 晴天.flac",
			wantMeta: &filenameMeta{artists: []string{"周杰伦"}, title: "晴天"},
		},
		{
			name:     "Alan Walker _ Iselin Solheim - Sing Me to Sleep.flac",
			wantMeta: &filenameMeta{artists: []string{"Alan Walker", "Iselin Solheim"}, title: "Sing Me to Sleep"},
		},
		{
			name:     "Christopher,Madcon - Limousine.flac",
			wantMeta: &filenameMeta{artists: []string{"Christopher", "Madcon"}, title: "Limousine"},
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
			wantMeta: &filenameMeta{title: "test1"},
		},
		{
			name:     "艺术家-标题格式（中文）",
			filename: "周杰伦 - 晴天.flac",
			wantMeta: &filenameMeta{artists: []string{"周杰伦"}, title: "晴天"},
		},
		{
			name:     "标题-艺术家格式（中文）",
			filename: "晴天 - 周杰伦.mflac",
			wantMeta: &filenameMeta{artists: []string{"周杰伦"}, title: "晴天"},
		},
		{
			name:     "艺术家-标题格式（英文）",
			filename: "Taylor Swift - Love Story.mp3",
			wantMeta: &filenameMeta{artists: []string{"Taylor Swift"}, title: "Love Story"},
		},
		{
			name:     "标题-艺术家格式（英文）",
			filename: "Love Story - Taylor Swift.mp3",
			wantMeta: &filenameMeta{artists: []string{"Taylor Swift"}, title: "Love Story"},
		},
		{
			name:     "包含Live的歌曲名",
			filename: "听妈妈的话 (Live) - 周杰伦.flac",
			wantMeta: &filenameMeta{artists: []string{"周杰伦"}, title: "听妈妈的话 (Live)"},
		},
		{
			name:     "多个艺术家",
			filename: "Alan Walker,Iselin Solheim - Sing Me to Sleep.flac",
			wantMeta: &filenameMeta{artists: []string{"Alan Walker", "Iselin Solheim"}, title: "Sing Me to Sleep"},
		},
		{
			name:     "复杂歌曲名",
			filename: "Shape of You (Ed Sheeran Cover) - 某艺术家.mp3",
			wantMeta: &filenameMeta{artists: []string{"某艺术家"}, title: "Shape of You (Ed Sheeran Cover)"},
		},
		{
			name:     "空文件名",
			filename: "",
			wantMeta: &filenameMeta{},
		},
		{
			name:     "只有扩展名",
			filename: ".mp3",
			wantMeta: &filenameMeta{},
		},
		{
			name:     "无分隔符",
			filename: "单个标题.mp3",
			wantMeta: &filenameMeta{title: "单个标题"},
		},
		{
			name:     "空的第一部分",
			filename: " - 周杰伦.mp3",
			wantMeta: &filenameMeta{title: "周杰伦"},
		},
		{
			name:     "空的第二部分",
			filename: "晴天 - .mp3",
			wantMeta: &filenameMeta{title: "晴天"},
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
