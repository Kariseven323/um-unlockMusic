package common

import (
	"path"
	"strings"
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
		return ParseFilenameMeta(filename)
	}

	filenameMeta := ParseFilenameMeta(filename)
	return &metaWrapper{
		original: original,
		filename: filenameMeta,
	}
}
