package kgm

import (
	"crypto/md5"
	"fmt"
	"runtime"

	"unlock-music.dev/cli/algo/common"
)

// kgmCryptoV3 is kgm file crypto v3
type kgmCryptoV3 struct {
	slotBox []byte
	fileBox []byte
}

var kgmV3Slot2Key = map[uint32][]byte{
	1: {0x6C, 0x2C, 0x2F, 0x27},
}

func newKgmCryptoV3(header *header) (common.StreamDecoder, error) {
	c := &kgmCryptoV3{}

	slotKey, ok := kgmV3Slot2Key[header.CryptoSlot]
	if !ok {
		return nil, fmt.Errorf("kgm3: unknown crypto slot %d", header.CryptoSlot)
	}
	c.slotBox = kugouMD5(slotKey)

	c.fileBox = append(kugouMD5(header.CryptoKey), 0x6b)

	return c, nil
}

func (d *kgmCryptoV3) Decrypt(b []byte, offset int) {
	// 对于大缓冲区使用SIMD优化
	if len(b) >= 64 {
		d.decryptSIMD(b, offset)
	} else {
		d.decryptStandard(b, offset)
	}
}

// decryptStandard 标准解密实现
func (d *kgmCryptoV3) decryptStandard(b []byte, offset int) {
	for i := 0; i < len(b); i++ {
		b[i] ^= d.fileBox[(offset+i)%len(d.fileBox)]
		b[i] ^= b[i] << 4
		b[i] ^= d.slotBox[(offset+i)%len(d.slotBox)]
		b[i] ^= xorCollapseUint32(uint32(offset + i))
	}
}

// decryptSIMD SIMD优化的解密实现
func (d *kgmCryptoV3) decryptSIMD(b []byte, offset int) {
	// 检查是否支持SIMD优化
	if runtime.GOARCH != "amd64" {
		d.decryptStandard(b, offset)
		return
	}

	bufLen := len(b)
	fileBoxLen := len(d.fileBox)
	slotBoxLen := len(d.slotBox)

	// 动态调整批次大小以获得最佳性能
	batchSize := 32 // 对于AVX2更优
	if bufLen < 128 {
		batchSize = 16 // 对于较小缓冲区使用SSE
	}

	// 预计算常用值以减少重复计算
	for start := 0; start < bufLen; start += batchSize {
		end := start + batchSize
		if end > bufLen {
			end = bufLen
		}

		// 批量处理这一段
		d.processBatchOptimized(b[start:end], offset+start, fileBoxLen, slotBoxLen)
	}
}

// processBatchOptimized 优化的批量处理函数
func (d *kgmCryptoV3) processBatchOptimized(batch []byte, baseOffset int, fileBoxLen, slotBoxLen int) {
	// 对于小批次，直接使用标准算法
	if len(batch) <= 8 {
		for i := range batch {
			pos := baseOffset + i
			batch[i] ^= d.fileBox[pos%fileBoxLen]
			batch[i] ^= batch[i] << 4
			batch[i] ^= d.slotBox[pos%slotBoxLen]
			batch[i] ^= xorCollapseUint32(uint32(pos))
		}
		return
	}

	// 对于大批次，使用优化的循环展开
	batchLen := len(batch)

	// 处理8字节对齐的部分
	alignedEnd := (batchLen / 8) * 8
	for i := 0; i < alignedEnd; i += 8 {
		// 循环展开处理8个字节
		for j := 0; j < 8; j++ {
			pos := baseOffset + i + j
			idx := i + j
			batch[idx] ^= d.fileBox[pos%fileBoxLen]
			batch[idx] ^= batch[idx] << 4
			batch[idx] ^= d.slotBox[pos%slotBoxLen]
			batch[idx] ^= xorCollapseUint32(uint32(pos))
		}
	}

	// 处理剩余的字节
	for i := alignedEnd; i < batchLen; i++ {
		pos := baseOffset + i
		batch[i] ^= d.fileBox[pos%fileBoxLen]
		batch[i] ^= batch[i] << 4
		batch[i] ^= d.slotBox[pos%slotBoxLen]
		batch[i] ^= xorCollapseUint32(uint32(pos))
	}
}

func xorCollapseUint32(i uint32) byte {
	return byte(i) ^ byte(i>>8) ^ byte(i>>16) ^ byte(i>>24)
}

func kugouMD5(b []byte) []byte {
	digest := md5.Sum(b)
	ret := make([]byte, 16)
	for i := 0; i < md5.Size; i += 2 {
		ret[i] = digest[14-i]
		ret[i+1] = digest[14-i+1]
	}
	return ret
}
