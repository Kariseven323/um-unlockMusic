package ncm

import (
	"sync"

	"unlock-music.dev/cli/internal/simd"
)

// ncmBoxPool NCM密钥box对象池
type ncmBoxPool struct {
	pool sync.Pool
}

// 全局NCM box池实例
var globalNCMBoxPool = &ncmBoxPool{
	pool: sync.Pool{
		New: func() interface{} {
			return make([]byte, 256)
		},
	},
}

// getBox 从池中获取box数组
func (p *ncmBoxPool) getBox() []byte {
	return p.pool.Get().([]byte)
}

// putBox 将box数组归还到池中
func (p *ncmBoxPool) putBox(box []byte) {
	if len(box) == 256 {
		// 清零以避免数据泄露
		for i := range box {
			box[i] = 0
		}
		p.pool.Put(box)
	}
}

type ncmCipher struct {
	key       []byte
	box       []byte
	optimized *simd.NCMCipherOptimized // SIMD优化实例
}

func newNcmCipher(key []byte) *ncmCipher {
	box := buildKeyBox(key)
	cipher := &ncmCipher{
		key:       key,
		box:       box,
		optimized: simd.NewNCMCipherOptimized(box),
	}
	return cipher
}

// Decrypt 解密方法，自动选择最优实现
func (c *ncmCipher) Decrypt(buf []byte, offset int) {
	// 对于大缓冲区使用SIMD优化
	if len(buf) >= 64 && c.optimized != nil {
		c.optimized.Decrypt(buf, offset)
	} else {
		// 小缓冲区使用标准方法
		c.decryptStandard(buf, offset)
	}
}

// decryptStandard 标准解密实现
func (c *ncmCipher) decryptStandard(buf []byte, offset int) {
	for i := 0; i < len(buf); i++ {
		buf[i] ^= c.box[(i+offset)&0xff]
	}
}

// buildKeyBox 构建密钥box，使用内存池优化
func buildKeyBox(key []byte) []byte {
	// 从池中获取临时box数组
	tempBox := globalNCMBoxPool.getBox()
	defer globalNCMBoxPool.putBox(tempBox)

	// 初始化临时box
	for i := 0; i < 256; i++ {
		tempBox[i] = byte(i)
	}

	// 第一轮混淆
	var j byte
	keyLen := len(key)
	for i := 0; i < 256; i++ {
		j = tempBox[i] + j + key[i%keyLen]
		tempBox[i], tempBox[j] = tempBox[j], tempBox[i]
	}

	// 生成最终的密钥box
	ret := make([]byte, 256)
	var _i byte
	for i := 0; i < 256; i++ {
		_i = byte(i + 1)
		si := tempBox[_i]
		sj := tempBox[_i+si]
		ret[i] = tempBox[si+sj]
	}
	return ret
}
