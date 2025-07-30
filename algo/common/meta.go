package common

import (
	"path"
	"slices"
	"strings"
	"unicode"
)

type filenameMeta struct {
	artists []string
	title   string
	album   string
}

func (f *filenameMeta) GetArtists() []string {
	return f.artists
}

func (f *filenameMeta) GetTitle() string {
	return f.title
}

func (f *filenameMeta) GetAlbum() string {
	return f.album
}

func ParseFilenameMeta(filename string) (meta AudioMeta) {
	partName := strings.TrimSuffix(filename, path.Ext(filename))
	items := strings.Split(partName, "-")
	ret := &filenameMeta{}

	switch len(items) {
	case 0:
		// no-op
	case 1:
		ret.title = strings.TrimSpace(items[0])
	default:
		// 第一部分是标题，后面的部分是艺术家
		ret.title = strings.TrimSpace(items[0])

		for _, v := range items[1:] {
			artists := strings.FieldsFunc(v, func(r rune) bool {
				return r == ',' || r == '_'
			})
			for _, artist := range artists {
				ret.artists = append(ret.artists, strings.TrimSpace(artist))
			}
		}
	}

	return ret
}

// metaWrapper 包装原始元数据，优先使用文件名中的标题
type metaWrapper struct {
	original AudioMeta
	filename AudioMeta
}

func (m *metaWrapper) GetTitle() string {
	// 优先使用文件名中的标题，因为它通常更准确
	if filenameTitle := m.filename.GetTitle(); filenameTitle != "" {
		return filenameTitle
	}
	return m.original.GetTitle()
}

func (m *metaWrapper) GetAlbum() string {
	// 专辑信息优先使用原始元数据
	if originalAlbum := m.original.GetAlbum(); originalAlbum != "" {
		return originalAlbum
	}
	return m.filename.GetAlbum()
}

func (m *metaWrapper) GetArtists() []string {
	// 艺术家信息优先使用原始元数据
	if originalArtists := m.original.GetArtists(); len(originalArtists) > 0 {
		return originalArtists
	}
	return m.filename.GetArtists()
}

// WrapMetaWithFilename 用文件名信息包装原始元数据，优化标题准确性
func WrapMetaWithFilename(original AudioMeta, filename string) AudioMeta {
	if original == nil {
		return SmartParseFilenameMeta(filename)
	}

	filenameMeta := SmartParseFilenameMeta(filename)
	return &metaWrapper{
		original: original,
		filename: filenameMeta,
	}
}

// SmartParseFilenameMeta 智能解析文件名元数据，支持多种命名格式
//
// 支持格式：
// 1. "艺术家 - 标题.ext" (如: "周杰伦 - 晴天.flac")
// 2. "标题 - 艺术家.ext" (如: "晴天 - 周杰伦.flac")
//
// 使用启发式算法自动识别格式：
// - 基于中英文艺术家名称模式
// - 基于歌曲标题特征（如包含Live、Remix等关键词）
// - 基于字符串长度和复杂度
// - 无法确定时默认使用"艺术家 - 标题"格式
//
// 该函数向后兼容，当解析失败时会回退到基本解析逻辑
func SmartParseFilenameMeta(filename string) AudioMeta {
	// 边界情况处理
	if filename == "" {
		return &filenameMeta{}
	}

	partName := strings.TrimSuffix(filename, path.Ext(filename))
	partName = strings.TrimSpace(partName)

	// 如果处理后为空，返回空元数据
	if partName == "" {
		return &filenameMeta{}
	}

	items := strings.Split(partName, "-")
	ret := &filenameMeta{}

	switch len(items) {
	case 0:
		// no-op
	case 1:
		ret.title = strings.TrimSpace(items[0])
	default:
		// 智能识别两个部分的角色
		part1 := strings.TrimSpace(items[0])
		part2 := strings.TrimSpace(strings.Join(items[1:], "-"))

		// 边界情况：如果任一部分为空，使用非空部分作为标题
		if part1 == "" && part2 != "" {
			ret.title = part2
			return ret
		} else if part2 == "" && part1 != "" {
			ret.title = part1
			return ret
		} else if part1 == "" && part2 == "" {
			return ret
		}

		// 快速路径：检查明显的艺术家特征
		if quickIdentifyArtist(part1) && !quickIdentifyArtist(part2) {
			// part1明显是艺术家且part2不是 (如: "周杰伦 - 晴天")
			ret.artists = []string{part1}
			ret.title = part2
		} else if quickIdentifyArtist(part2) && !quickIdentifyArtist(part1) {
			// part2明显是艺术家且part1不是 (如: "晴天 - 周杰伦")
			ret.title = part1
			ret.artists = []string{part2}
		} else if isLikelyArtistName(part1) && isLikelySongTitle(part2) {
			// 启发式判断：艺术家 - 标题
			ret.artists = []string{part1}
			ret.title = part2
		} else if isLikelySongTitle(part1) && isLikelyArtistName(part2) {
			// 启发式判断：标题 - 艺术家
			ret.title = part1
			ret.artists = []string{part2}
		} else {
			// 无法确定，使用默认格式（艺术家 - 标题）
			ret.artists = []string{part1}
			ret.title = part2
		}

		// 处理多个艺术家的情况
		if len(ret.artists) > 0 {
			var allArtists []string
			for _, artist := range ret.artists {
				artists := strings.FieldsFunc(artist, func(r rune) bool {
					return r == ',' || r == '_'
				})
				for _, a := range artists {
					allArtists = append(allArtists, strings.TrimSpace(a))
				}
			}
			ret.artists = allArtists
		}
	}

	return ret
}

// quickIdentifyArtist 快速识别明显的艺术家名称特征
// 返回true表示该字符串明显是艺术家名称，可以跳过复杂的启发式分析
func quickIdentifyArtist(name string) bool {
	if name == "" {
		return false
	}

	// 中文艺术家：包含常见姓氏且长度合适
	if isChinese(name) {
		runeCount := len([]rune(name))
		if runeCount >= 2 && runeCount <= 4 && hasCommonChineseSurname(name) {
			return true
		}
	}

	// 英文艺术家：首字母大写且包含空格（如"Taylor Swift"）
	// 但要确保不是歌曲名（如"Love Story"这样的短语）
	if isEnglish(name) && isCapitalized(name) && strings.Contains(name, " ") {
		// 检查是否包含明显的歌曲关键词，如果有则不是艺术家
		if !containsSongKeywords(name) {
			// 进一步检查：如果是常见的歌曲名模式，则不认为是艺术家
			words := strings.Fields(name)
			if len(words) == 2 {
				// 检查是否是常见的歌曲名模式（如"Love Story", "Bad Romance"等）
				// 但"Taylor Swift", "John Doe"等明显是人名
				word1, word2 := words[0], words[1]

				// 如果第二个词是常见的歌曲词汇，可能是歌曲名
				songWords := []string{"Story", "Song", "Dream", "Night", "Day", "Love", "Heart", "Life", "Time", "World"}
				if slices.Contains(songWords, word2) {
					return false // 可能是歌曲名
				}

				// 如果都是短词且没有明显的人名特征，可能是歌曲名
				if len(word1) <= 5 && len(word2) <= 5 {
					return false
				}
			}
			return true
		}
	}

	return false
}

// isLikelyArtistName 判断字符串是否更像艺术家名称
func isLikelyArtistName(name string) bool {
	if name == "" {
		return false
	}

	// 计算艺术家名称的可能性分数
	score := 0

	// 中文艺术家名称模式
	if isChinese(name) {
		runeCount := len([]rune(name))
		// 中文艺术家名通常2-4个字符
		if runeCount >= 2 && runeCount <= 4 {
			score += 3
		}
		// 常见中文姓氏是强指标
		if hasCommonChineseSurname(name) {
			score += 4
		}
	}

	// 英文艺术家名称模式
	if isEnglish(name) {
		// 首字母大写
		if isCapitalized(name) {
			score += 2
		}
		// 包含空格（如 "Taylor Swift"）是强指标，但要排除歌曲名
		if strings.Contains(name, " ") && !containsSongKeywords(name) {
			score += 3
		}
		// 长度因素：英文艺术家名通常较短
		if len(name) <= 15 {
			score += 1
		}
		// 如果是单个英文词且首字母大写，可能是艺术家
		if !strings.Contains(name, " ") && isCapitalized(name) && len(name) <= 10 {
			score += 1
		}
	}

	// 不包含特殊符号（歌曲名更可能包含括号等）
	if !containsSpecialChars(name) {
		score += 1
	}

	return score >= 4
}

// isLikelySongTitle 判断字符串是否更像歌曲标题
func isLikelySongTitle(title string) bool {
	if title == "" {
		return false
	}

	// 计算歌曲标题的可能性分数
	score := 0

	// 包含特殊符号（如括号、数字等）是强指标
	if containsSpecialChars(title) {
		score += 4
	}

	// 包含常见歌曲标识词是强指标
	if containsSongKeywords(title) {
		score += 5
	}

	// 包含数字
	if containsNumbers(title) {
		score += 2
	}

	// 长度因素：歌曲名通常较长
	runeCount := len([]rune(title))
	if runeCount > 6 {
		score += 2
	}
	if runeCount > 10 {
		score += 1
	}

	// 中文歌曲名通常比艺术家名长
	if isChinese(title) && runeCount > 4 {
		score += 2
	}

	return score >= 3
}

// 辅助函数：字符类型判断

// isChinese 判断字符串是否主要包含中文字符
func isChinese(s string) bool {
	chineseCount := 0
	totalCount := 0
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			chineseCount++
		}
		if !unicode.IsSpace(r) {
			totalCount++
		}
	}
	return totalCount > 0 && float64(chineseCount)/float64(totalCount) > 0.5
}

// isEnglish 判断字符串是否主要包含英文字符
func isEnglish(s string) bool {
	englishCount := 0
	totalCount := 0
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			englishCount++
		}
		if !unicode.IsSpace(r) {
			totalCount++
		}
	}
	return totalCount > 0 && float64(englishCount)/float64(totalCount) > 0.5
}

// isCapitalized 判断字符串是否首字母大写
func isCapitalized(s string) bool {
	if len(s) == 0 {
		return false
	}
	words := strings.Fields(s)
	for _, word := range words {
		if len(word) > 0 && !unicode.IsUpper([]rune(word)[0]) {
			return false
		}
	}
	return len(words) > 0
}

// hasCommonChineseSurname 检查是否包含常见中文姓氏
func hasCommonChineseSurname(name string) bool {
	if len([]rune(name)) == 0 {
		return false
	}

	firstChar := string([]rune(name)[0])
	return slices.Contains(commonChineseSurnames, firstChar)
}

// containsSpecialChars 检查字符串是否包含特殊字符
func containsSpecialChars(s string) bool {
	specialChars := []string{"(", ")", "[", "]", "{", "}", "（", "）", "【", "】"}
	for _, char := range specialChars {
		if strings.Contains(s, char) {
			return true
		}
	}
	return false
}

// 预编译的常量列表，提高性能
var (
	// 常见歌曲关键词
	songKeywords = []string{
		"Live", "live", "LIVE",
		"Remix", "remix", "REMIX",
		"Cover", "cover", "COVER",
		"Acoustic", "acoustic", "ACOUSTIC",
		"Instrumental", "instrumental", "INSTRUMENTAL",
		"Demo", "demo", "DEMO",
		"Version", "version", "VERSION",
		"Mix", "mix", "MIX",
		"现场", "翻唱", "伴奏", "纯音乐", "演奏版",
	}

	// 常见中文姓氏
	commonChineseSurnames = []string{
		"王", "李", "张", "刘", "陈", "杨", "黄", "赵", "周", "吴",
		"徐", "孙", "朱", "马", "胡", "郭", "林", "何", "高", "梁",
		"郑", "罗", "宋", "谢", "唐", "韩", "曹", "许", "邓", "萧",
	}
)

// containsSongKeywords 检查是否包含常见歌曲关键词
func containsSongKeywords(s string) bool {
	for _, keyword := range songKeywords {
		if strings.Contains(s, keyword) {
			return true
		}
	}
	return false
}

// containsNumbers 检查字符串是否包含数字
func containsNumbers(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}
