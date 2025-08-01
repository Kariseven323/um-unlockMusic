package common

import (
	"path"
	"slices"
	"strings"
	"unicode"
)

type filenameMeta struct {
	artists        []string
	title          string
	album          string
	originalFormat string
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

func (f *filenameMeta) GetOriginalFormat() string {
	return f.originalFormat
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

func (m *metaWrapper) GetOriginalFormat() string {
	// 原始格式信息优先使用文件名中的格式
	return m.filename.GetOriginalFormat()
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

	// 预处理：如果包含 "_" 分隔符，只取第一部分进行解析
	// 这样可以排除额外信息的干扰，专注于 "歌曲名 - 歌手名" 部分的识别
	if strings.Contains(partName, "_") {
		underscoreParts := strings.Split(partName, "_")
		if len(underscoreParts) > 0 && strings.TrimSpace(underscoreParts[0]) != "" {
			partName = strings.TrimSpace(underscoreParts[0])
		}
	}

	items := strings.Split(partName, "-")
	ret := &filenameMeta{}

	switch len(items) {
	case 0:
		// no-op
	case 1:
		ret.title = strings.TrimSpace(items[0])
		ret.originalFormat = "title-only"
	default:
		// 智能识别两个部分的角色
		part1 := strings.TrimSpace(items[0])
		part2 := strings.TrimSpace(strings.Join(items[1:], "-"))

		// 边界情况：如果任一部分为空，使用非空部分作为标题
		if part1 == "" && part2 != "" {
			ret.title = part2
			ret.originalFormat = "title-only"
			return ret
		} else if part2 == "" && part1 != "" {
			ret.title = part1
			ret.originalFormat = "title-only"
			return ret
		} else if part1 == "" && part2 == "" {
			ret.originalFormat = "empty"
			return ret
		}

		// 使用新的多语言智能分析
		isArtistTitle, confidence := analyzeByLanguage(part1, part2)

		// 如果置信度很高，直接使用分析结果
		if confidence > 0.7 {
			if isArtistTitle {
				ret.artists = []string{removeQualitySuffix(part1)}
				ret.title = removeQualitySuffix(part2)
				ret.originalFormat = "artist-title"
			} else {
				ret.title = removeQualitySuffix(part1)
				ret.artists = []string{removeQualitySuffix(part2)}
				ret.originalFormat = "title-artist"
			}
		} else {
			// 置信度不高时，使用传统启发式作为备选
			// 快速路径：检查明显的艺术家特征
			cleanPart1 := removeQualitySuffix(part1)
			cleanPart2 := removeQualitySuffix(part2)

			if quickIdentifyArtist(cleanPart1) && !quickIdentifyArtist(cleanPart2) {
				// part1明显是艺术家且part2不是
				ret.artists = []string{cleanPart1}
				ret.title = cleanPart2
				ret.originalFormat = "artist-title"
			} else if quickIdentifyArtist(cleanPart2) && !quickIdentifyArtist(cleanPart1) {
				// part2明显是艺术家且part1不是
				ret.title = cleanPart1
				ret.artists = []string{cleanPart2}
				ret.originalFormat = "title-artist"
			} else if isLikelyArtistName(cleanPart1) && isLikelySongTitle(cleanPart2) {
				// 启发式判断：艺术家 - 标题
				ret.artists = []string{cleanPart1}
				ret.title = cleanPart2
				ret.originalFormat = "artist-title"
			} else if isLikelySongTitle(cleanPart1) && isLikelyArtistName(cleanPart2) {
				// 启发式判断：标题 - 艺术家
				ret.title = cleanPart1
				ret.artists = []string{cleanPart2}
				ret.originalFormat = "title-artist"
			} else {
				// 无法确定时，优先考虑多语言分析结果
				if isArtistTitle {
					ret.artists = []string{cleanPart1}
					ret.title = cleanPart2
					ret.originalFormat = "artist-title"
				} else {
					ret.title = cleanPart1
					ret.artists = []string{cleanPart2}
					ret.originalFormat = "title-artist"
				}
			}
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
	// 常见歌曲关键词（多语言）
	songKeywords = []string{
		// 英文
		"Live", "live", "LIVE",
		"Remix", "remix", "REMIX",
		"Cover", "cover", "COVER",
		"Acoustic", "acoustic", "ACOUSTIC",
		"Instrumental", "instrumental", "INSTRUMENTAL",
		"Demo", "demo", "DEMO",
		"Version", "version", "VERSION",
		"Mix", "mix", "MIX",
		"Remaster", "remaster", "REMASTER",
		"Extended", "extended", "EXTENDED",
		"Radio", "radio", "RADIO",
		"Edit", "edit", "EDIT",
		// 中文
		"现场", "翻唱", "伴奏", "纯音乐", "演奏版", "重制版", "混音版",
		"电台版", "完整版", "精选版", "特别版", "原声版",
		// 日文
		"ライブ", "リミックス", "カバー", "アコースティック", "インストゥルメンタル",
		"デモ", "バージョン", "ミックス", "リマスター",
		// 韩文
		"라이브", "리믹스", "커버", "어쿠스틱", "인스트루멘탈",
		"데모", "버전", "믹스", "리마스터",
	}

	// 音质标识后缀
	qualitySuffixes = []string{
		"_hires", "_HIRES", "_HiRes",
		"_live", "_LIVE", "_Live",
		"_lossless", "_LOSSLESS", "_Lossless",
		"_flac", "_FLAC", "_Flac",
		"_dsd", "_DSD", "_Dsd",
		"_24bit", "_24BIT", "_24Bit",
		"_96khz", "_96KHZ", "_96kHz",
		"_192khz", "_192KHZ", "_192kHz",
		"_studio", "_STUDIO", "_Studio",
		"_master", "_MASTER", "_Master",
		"_remaster", "_REMASTER", "_Remaster",
		"_original", "_ORIGINAL", "_Original",
		"_deluxe", "_DELUXE", "_Deluxe",
		"_special", "_SPECIAL", "_Special",
		"_edition", "_EDITION", "_Edition",
		"_version", "_VERSION", "_Version",
	}

	// 常见中文姓氏
	commonChineseSurnames = []string{
		"王", "李", "张", "刘", "陈", "杨", "黄", "赵", "周", "吴",
		"徐", "孙", "朱", "马", "胡", "郭", "林", "何", "高", "梁",
		"郑", "罗", "宋", "谢", "唐", "韩", "曹", "许", "邓", "萧",
		"蒋", "沈", "韩", "杨", "朱", "秦", "尤", "许", "何", "吕",
		"施", "张", "孔", "曹", "严", "华", "金", "魏", "陶", "姜",
	}

	// 常见日文姓氏
	commonJapaneseSurnames = []string{
		"田", "中", "佐", "藤", "山", "木", "村", "井", "上", "野",
		"川", "松", "本", "小", "林", "高", "橋", "渡", "辺", "伊",
		"加", "藤", "森", "石", "田", "前", "田", "近", "藤", "坂",
	}

	// 常见韩文姓氏
	commonKoreanSurnames = []string{
		"김", "이", "박", "최", "정", "강", "조", "윤", "장", "임",
		"한", "오", "서", "신", "권", "황", "안", "송", "류", "전",
		"홍", "고", "문", "양", "손", "배", "조", "백", "허", "유",
		"노", "심", "원", "민", "성", "곽", "변", "남", "진", "어",
		"엄", "채", "원", "천", "방", "공", "강", "현", "함", "변",
		"염", "양", "변", "여", "추", "노", "도", "소", "신", "석",
		"선", "설", "마", "길", "주", "연", "방", "위", "표", "명",
		"기", "반", "왕", "금", "옥", "육", "인", "맹", "제", "모",
		"장", "남", "탁", "국", "여", "진", "어", "은", "편", "구",
		"용", "구", "갈", "등", "정", "좌", "승", "사", "마", "서",
		// 특별한 경우: 싸이 (PSY)는 예명이므로 특별 처리
		"싸",
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

// LanguageType 表示检测到的语言类型
type LanguageType int

const (
	LanguageUnknown LanguageType = iota
	LanguageChinese
	LanguageEnglish
	LanguageJapanese
	LanguageKorean
	LanguageRussian
	LanguageMixed
)

// removeQualitySuffix 去除音质标识后缀
func removeQualitySuffix(name string) string {
	result := name
	// 循环去除所有可能的后缀，直到没有更多后缀可以去除
	for {
		originalResult := result
		for _, suffix := range qualitySuffixes {
			if strings.HasSuffix(result, suffix) {
				result = strings.TrimSpace(strings.TrimSuffix(result, suffix))
				break
			}
		}
		// 如果这一轮没有去除任何后缀，则停止
		if result == originalResult {
			break
		}
	}
	return result
}

// detectLanguage 检测字符串的主要语言类型
func detectLanguage(s string) LanguageType {
	if s == "" {
		return LanguageUnknown
	}

	chineseCount := 0
	englishCount := 0
	japaneseCount := 0
	koreanCount := 0
	russianCount := 0
	totalCount := 0

	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsDigit(r) {
			continue
		}
		totalCount++

		// 中文字符 (CJK统一汉字)
		if unicode.Is(unicode.Han, r) {
			chineseCount++
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			// 英文字符
			englishCount++
		} else if (r >= 0x3040 && r <= 0x309F) || (r >= 0x30A0 && r <= 0x30FF) {
			// 日文字符 (平假名和片假名)
			japaneseCount++
		} else if r >= 0xAC00 && r <= 0xD7AF {
			// 韩文字符 (한글)
			koreanCount++
		} else if (r >= 0x0400 && r <= 0x04FF) || (r >= 0x0500 && r <= 0x052F) {
			// 俄文字符 (西里尔字母)
			russianCount++
		}
	}

	if totalCount == 0 {
		return LanguageUnknown
	}

	// 计算各语言占比
	chineseRatio := float64(chineseCount) / float64(totalCount)
	englishRatio := float64(englishCount) / float64(totalCount)
	japaneseRatio := float64(japaneseCount) / float64(totalCount)
	koreanRatio := float64(koreanCount) / float64(totalCount)
	russianRatio := float64(russianCount) / float64(totalCount)

	// 混合语言检测 (任意两种语言占比都超过20%)
	languageCount := 0
	if chineseRatio > 0.2 {
		languageCount++
	}
	if englishRatio > 0.2 {
		languageCount++
	}
	if japaneseRatio > 0.2 {
		languageCount++
	}
	if koreanRatio > 0.2 {
		languageCount++
	}
	if russianRatio > 0.2 {
		languageCount++
	}

	if languageCount > 1 {
		return LanguageMixed
	}

	// 单一语言检测 (占比超过50%)
	if chineseRatio > 0.5 {
		return LanguageChinese
	}
	if englishRatio > 0.5 {
		return LanguageEnglish
	}
	if japaneseRatio > 0.5 {
		return LanguageJapanese
	}
	if koreanRatio > 0.5 {
		return LanguageKorean
	}
	if russianRatio > 0.5 {
		return LanguageRussian
	}

	// 如果没有明显的主导语言，选择占比最高的
	maxRatio := chineseRatio
	result := LanguageChinese

	if englishRatio > maxRatio {
		maxRatio = englishRatio
		result = LanguageEnglish
	}
	if japaneseRatio > maxRatio {
		maxRatio = japaneseRatio
		result = LanguageJapanese
	}
	if koreanRatio > maxRatio {
		maxRatio = koreanRatio
		result = LanguageKorean
	}
	if russianRatio > maxRatio {
		maxRatio = russianRatio
		result = LanguageRussian
	}

	// 如果最高占比仍然很低，返回未知
	if maxRatio < 0.3 {
		return LanguageUnknown
	}

	return result
}

// getSongTitlePatterns 获取特定语言的歌曲名模式特征
func getSongTitlePatterns(lang LanguageType) map[string]int {
	patterns := make(map[string]int)

	switch lang {
	case LanguageChinese:
		// 中文歌曲名常见模式
		chinesePatterns := map[string]int{
			// 地名相关
			"北京": 3, "上海": 3, "广州": 3, "深圳": 3, "杭州": 3,
			"南京": 3, "西安": 3, "成都": 3, "重庆": 3, "天津": 3,
			"香港": 3, "台北": 3, "澳门": 3,
			// 情感词汇
			"爱情": 4, "思念": 4, "回忆": 4, "梦想": 4, "青春": 4,
			"孤独": 4, "寂寞": 4, "温柔": 4, "浪漫": 4, "甜蜜": 4,
			"心痛": 4, "眼泪": 4, "微笑": 4, "拥抱": 4, "告别": 4,
			// 时间词汇
			"昨天": 3, "今天": 3, "明天": 3, "永远": 3, "瞬间": 3,
			"春天": 3, "夏天": 3, "秋天": 3, "冬天": 3, "夜晚": 3,
			"黎明": 3, "黄昏": 3, "午夜": 3,
			// 颜色词汇
			"红色": 3, "蓝色": 3, "白色": 3, "黑色": 3, "绿色": 3,
			"紫色": 3, "黄色": 3, "粉色": 3, "灰色": 3,
			// 自然元素
			"月亮": 3, "太阳": 3, "星星": 3, "海洋": 3, "山峰": 3,
			"花朵": 3, "树叶": 3, "雨水": 3, "雪花": 3, "风景": 3,
			// 抽象概念
			"自由": 3, "希望": 3, "信念": 3, "勇气": 3, "力量": 3,
			"奇迹": 3, "命运": 3, "缘分": 3, "幸福": 3, "快乐": 3,
		}
		for k, v := range chinesePatterns {
			patterns[k] = v
		}

	case LanguageEnglish:
		// 英文歌曲名常见模式
		englishPatterns := map[string]int{
			// 情感词汇
			"love": 4, "heart": 4, "dream": 4, "hope": 4, "life": 4,
			"time": 4, "night": 4, "day": 4, "light": 4, "dark": 4,
			"soul": 4, "mind": 4, "eyes": 4, "smile": 4, "tears": 4,
			"kiss": 4, "touch": 4, "hold": 4, "feel": 4, "miss": 4,
			// 动作词汇
			"dance": 3, "sing": 3, "fly": 3, "run": 3, "walk": 3,
			"fall": 3, "rise": 3, "shine": 3, "burn": 3, "break": 3,
			// 自然元素
			"moon": 3, "sun": 3, "star": 3, "sky": 3, "sea": 3,
			"fire": 3, "water": 3, "wind": 3, "rain": 3, "snow": 3,
			// 抽象概念
			"freedom": 3, "peace": 3, "power": 3, "magic": 3, "wonder": 3,
			"miracle": 3, "destiny": 3, "forever": 3, "always": 3, "never": 3,
		}
		for k, v := range englishPatterns {
			patterns[k] = v
		}

	case LanguageJapanese:
		// 日文歌曲名常见模式
		japanesePatterns := map[string]int{
			// 情感词汇
			"愛": 4, "恋": 4, "心": 4, "夢": 4, "希望": 4,
			"涙": 4, "笑顔": 4, "想い": 4, "気持ち": 4, "感情": 4,
			// 时间词汇
			"今日": 3, "明日": 3, "昨日": 3, "永遠": 3, "瞬間": 3,
			"春": 3, "夏": 3, "秋": 3, "冬": 3, "夜": 3,
			// 自然元素
			"月": 3, "太陽": 3, "星": 3, "海": 3, "空": 3,
			"花": 3, "桜": 3, "雨": 3, "雪": 3, "風": 3,
			// 抽象概念
			"自由": 3, "平和": 3, "力": 3, "魔法": 3, "奇跡": 3,
		}
		for k, v := range japanesePatterns {
			patterns[k] = v
		}

	case LanguageKorean:
		// 韩文歌曲名常见模式
		koreanPatterns := map[string]int{
			// 情感词汇
			"사랑": 4, "마음": 4, "꿈": 4, "희망": 4, "기억": 4,
			"눈물": 4, "미소": 4, "그리움": 4, "행복": 4, "슬픔": 4,
			// 时间词汇
			"오늘": 3, "내일": 3, "어제": 3, "영원": 3, "순간": 3,
			"봄": 3, "여름": 3, "가을": 3, "겨울": 3, "밤": 3,
			// 自然元素
			"달": 3, "해": 3, "별": 3, "바다": 3, "하늘": 3,
			"꽃": 3, "나무": 3, "비": 3, "눈": 3, "바람": 3,
			// 抽象概念
			"자유": 3, "평화": 3, "힘": 3, "기적": 3, "운명": 3,
		}
		for k, v := range koreanPatterns {
			patterns[k] = v
		}
	}

	return patterns
}

// getArtistNamePatterns 获取特定语言的艺术家名模式特征
func getArtistNamePatterns(lang LanguageType) map[string]int {
	patterns := make(map[string]int)

	switch lang {
	case LanguageChinese:
		// 中文艺术家名常见模式
		chinesePatterns := map[string]int{
			// 常见艺术家名后缀
			"组合": 3, "乐队": 3, "乐团": 3, "合唱团": 3, "工作室": 3,
			"音乐": 2, "歌手": 2, "艺人": 2, "明星": 2,
			// 常见艺术家名前缀
			"小": 2, "大": 2, "老": 2, "阿": 2,
		}
		for k, v := range chinesePatterns {
			patterns[k] = v
		}

	case LanguageEnglish:
		// 英文艺术家名常见模式
		englishPatterns := map[string]int{
			// 乐队/组合后缀
			"band": 3, "group": 3, "crew": 3, "collective": 3,
			"orchestra": 3, "ensemble": 3, "choir": 3, "quartet": 3,
			"trio": 3, "duo": 3, "brothers": 3, "sisters": 3,
			// 常见艺术家名词汇
			"mc": 2, "dj": 2, "dr": 2, "mr": 2, "ms": 2,
			"the": 1, "and": 1, "of": 1, "for": 1,
		}
		for k, v := range englishPatterns {
			patterns[k] = v
		}

	case LanguageJapanese:
		// 日文艺术家名常见模式
		japanesePatterns := map[string]int{
			// 乐队/组合后缀
			"バンド": 3, "グループ": 3, "ユニット": 3, "チーム": 3,
			"楽団": 3, "合唱団": 3, "オーケストラ": 3,
			// 常见艺术家名词汇
			"さん": 2, "ちゃん": 2, "くん": 2, "様": 2,
		}
		for k, v := range japanesePatterns {
			patterns[k] = v
		}

	case LanguageKorean:
		// 韩文艺术家名常见模式
		koreanPatterns := map[string]int{
			// 乐队/组合后缀
			"밴드": 3, "그룹": 3, "팀": 3, "유닛": 3,
			"오케스트라": 3, "합창단": 3, "앙상블": 3,
			// 常见艺术家名词汇
			"씨": 2, "님": 2, "군": 2, "양": 2,
		}
		for k, v := range koreanPatterns {
			patterns[k] = v
		}
	}

	return patterns
}

// hasCommonSurnameByLanguage 检查是否包含特定语言的常见姓氏
func hasCommonSurnameByLanguage(name string, lang LanguageType) bool {
	if len([]rune(name)) == 0 {
		return false
	}

	firstChar := string([]rune(name)[0])

	switch lang {
	case LanguageChinese:
		return slices.Contains(commonChineseSurnames, firstChar)
	case LanguageJapanese:
		return slices.Contains(commonJapaneseSurnames, firstChar)
	case LanguageKorean:
		return slices.Contains(commonKoreanSurnames, firstChar)
	default:
		return false
	}
}

// analyzeByLanguage 基于语言类型进行智能分析
func analyzeByLanguage(part1, part2 string) (isArtistTitle bool, confidence float64) {
	// 去除音质标识后缀
	cleanPart1 := removeQualitySuffix(part1)
	cleanPart2 := removeQualitySuffix(part2)

	// 检测两个部分的语言类型
	lang1 := detectLanguage(cleanPart1)
	lang2 := detectLanguage(cleanPart2)

	// 获取语言特定的评分
	artistScore1 := getLanguageSpecificScore(cleanPart1, lang1, true) // 作为艺术家的评分
	songScore1 := getLanguageSpecificScore(cleanPart1, lang1, false)  // 作为歌曲的评分
	artistScore2 := getLanguageSpecificScore(cleanPart2, lang2, true) // 作为艺术家的评分
	songScore2 := getLanguageSpecificScore(cleanPart2, lang2, false)  // 作为歌曲的评分

	// 计算两种格式的总分
	artistTitleScore := artistScore1 + songScore2 // part1是艺术家，part2是歌曲
	titleArtistScore := songScore1 + artistScore2 // part1是歌曲，part2是艺术家

	// 确定最佳格式
	if artistTitleScore > titleArtistScore {
		confidence = artistTitleScore / (artistTitleScore + titleArtistScore)
		return true, confidence
	} else {
		confidence = titleArtistScore / (artistTitleScore + titleArtistScore)
		return false, confidence
	}
}

// getLanguageSpecificScore 获取语言特定的评分
func getLanguageSpecificScore(text string, lang LanguageType, isArtist bool) float64 {
	if text == "" {
		return 0.0
	}

	score := 0.0
	cleanText := removeQualitySuffix(text)
	runeCount := len([]rune(cleanText))

	if isArtist {
		// 艺术家名评分
		switch lang {
		case LanguageChinese:
			// 中文艺术家名特征
			if runeCount >= 2 && runeCount <= 4 {
				score += 3.0
			}
			if hasCommonSurnameByLanguage(cleanText, lang) {
				score += 4.0
			}
			// 检查艺术家名模式
			patterns := getArtistNamePatterns(lang)
			for pattern, weight := range patterns {
				if strings.Contains(cleanText, pattern) {
					score += float64(weight)
				}
			}

		case LanguageEnglish:
			// 英文艺术家名特征
			if isCapitalized(cleanText) {
				score += 2.0
			}
			if strings.Contains(cleanText, " ") && !containsSongKeywords(cleanText) {
				score += 3.0
			}
			if runeCount <= 15 {
				score += 1.0
			}

			// 特殊处理：短的英文艺术家名（如 U2, AC/DC 等）
			if runeCount <= 4 && isCapitalized(cleanText) && !strings.Contains(cleanText, " ") {
				// 检查是否包含数字或特殊字符（常见于乐队名）
				if containsNumbers(cleanText) || strings.ContainsAny(cleanText, "/\\&") {
					score += 4.0
				}
				// 全大写的短名称通常是艺术家名
				if strings.ToUpper(cleanText) == cleanText {
					score += 3.0
				}
			}

			// 检查艺术家名模式
			patterns := getArtistNamePatterns(lang)
			lowerText := strings.ToLower(cleanText)
			for pattern, weight := range patterns {
				if strings.Contains(lowerText, pattern) {
					score += float64(weight)
				}
			}

		case LanguageJapanese:
			// 日文艺术家名特征
			if runeCount >= 2 && runeCount <= 6 {
				score += 2.0
			}
			if hasCommonSurnameByLanguage(cleanText, lang) {
				score += 3.0
			}
			// 检查艺术家名模式
			patterns := getArtistNamePatterns(lang)
			for pattern, weight := range patterns {
				if strings.Contains(cleanText, pattern) {
					score += float64(weight)
				}
			}

		case LanguageKorean:
			// 韩文艺术家名特征
			if runeCount >= 2 && runeCount <= 5 {
				score += 2.0
			}
			if hasCommonSurnameByLanguage(cleanText, lang) {
				score += 3.0
			}
			// 检查艺术家名模式
			patterns := getArtistNamePatterns(lang)
			for pattern, weight := range patterns {
				if strings.Contains(cleanText, pattern) {
					score += float64(weight)
				}
			}

		default:
			// 通用艺术家名特征
			if isCapitalized(cleanText) {
				score += 1.0
			}
			if runeCount >= 2 && runeCount <= 20 {
				score += 1.0
			}
		}

		// 通用艺术家名特征
		if !containsSpecialChars(cleanText) {
			score += 1.0
		}
		if !containsSongKeywords(cleanText) {
			score += 1.0
		}

	} else {
		// 歌曲名评分
		if containsSpecialChars(cleanText) {
			score += 4.0
		}
		if containsSongKeywords(cleanText) {
			score += 5.0
		}
		if containsNumbers(cleanText) {
			score += 2.0
		}
		if runeCount > 6 {
			score += 2.0
		}
		if runeCount > 10 {
			score += 1.0
		}

		// 检查歌曲名模式
		patterns := getSongTitlePatterns(lang)
		lowerText := strings.ToLower(cleanText)
		for pattern, weight := range patterns {
			if strings.Contains(lowerText, pattern) || strings.Contains(cleanText, pattern) {
				score += float64(weight)
			}
		}

		// 语言特定的歌曲名特征
		switch lang {
		case LanguageChinese:
			if runeCount > 4 {
				score += 2.0
			}
		case LanguageJapanese:
			if runeCount > 3 {
				score += 1.5
			}
		case LanguageKorean:
			if runeCount > 3 {
				score += 1.5
			}
		}
	}

	return score
}
