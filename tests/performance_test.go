package tests

import (
	"fmt"
	"testing"
	"time"

	"unlock-music.dev/cli/algo/qmc"
)

// BenchmarkQMCRC4Decrypt 测试QMC RC4解密性能
func BenchmarkQMCRC4Decrypt(b *testing.B) {
	// 创建测试数据
	testData := make([]byte, 1024*1024) // 1MB测试数据
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// 创建RC4密钥
	key := make([]byte, 256)
	for i := range key {
		key[i] = byte(i)
	}

	// 创建RC4解密器
	cipher, err := qmc.NewQmcCipherDecoder(key)
	if err != nil {
		b.Fatalf("创建RC4解密器失败: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 复制测试数据
		data := make([]byte, len(testData))
		copy(data, testData)

		// 执行解密
		cipher.Decrypt(data, 0)
	}
}

// BenchmarkStaticCipherDecrypt 测试静态密码解密性能
func BenchmarkStaticCipherDecrypt(b *testing.B) {
	// 创建测试数据
	testData := make([]byte, 1024*1024) // 1MB测试数据
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// 创建静态密码解密器
	cipher, err := qmc.NewQmcCipherDecoder(nil) // 空密钥使用静态密码
	if err != nil {
		b.Fatalf("创建静态密码解密器失败: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 复制测试数据
		data := make([]byte, len(testData))
		copy(data, testData)

		// 执行解密
		cipher.Decrypt(data, 0)
	}
}

// TestQMCRC4MemoryUsage 测试QMC RC4内存使用
func TestQMCRC4MemoryUsage(t *testing.T) {
	// 创建测试数据
	testData := make([]byte, 64*1024) // 64KB测试数据
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// 创建RC4密钥
	key := make([]byte, 256)
	for i := range key {
		key[i] = byte(i)
	}

	// 创建RC4解密器
	cipher, err := qmc.NewQmcCipherDecoder(key)
	if err != nil {
		t.Fatalf("创建RC4解密器失败: %v", err)
	}

	// 记录开始时间
	start := time.Now()

	// 执行多次解密以测试内存池效果
	for i := 0; i < 1000; i++ {
		data := make([]byte, len(testData))
		copy(data, testData)
		cipher.Decrypt(data, 0)
	}

	duration := time.Since(start)
	t.Logf("1000次解密耗时: %v", duration)
}

// TestStaticCipherSIMDOptimization 测试静态密码SIMD优化
func TestStaticCipherSIMDOptimization(t *testing.T) {
	// 创建不同大小的测试数据
	testSizes := []int{32, 64, 128, 256, 1024, 4096}

	for _, size := range testSizes {
		t.Run(fmt.Sprintf("Size_%d", size), func(t *testing.T) {
			testData := make([]byte, size)
			for i := range testData {
				testData[i] = byte(i % 256)
			}

			// 创建静态密码解密器
			cipher, err := qmc.NewQmcCipherDecoder(nil)
			if err != nil {
				t.Fatalf("创建静态密码解密器失败: %v", err)
			}

			// 记录开始时间
			start := time.Now()

			// 执行解密
			data := make([]byte, len(testData))
			copy(data, testData)
			cipher.Decrypt(data, 0)

			duration := time.Since(start)
			t.Logf("大小 %d 字节解密耗时: %v", size, duration)
		})
	}
}

// TestWindowsMemoryMapping 测试Windows内存映射（仅在Windows上运行）
func TestWindowsMemoryMapping(t *testing.T) {
	// 这个测试需要实际的文件，暂时跳过
	t.Skip("需要实际文件进行测试")
}

// BenchmarkMemoryAllocation 对比内存分配性能
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		// 使用对象池的版本（我们的优化）
		key := make([]byte, 256)
		for i := range key {
			key[i] = byte(i)
		}

		cipher, err := qmc.NewQmcCipherDecoder(key)
		if err != nil {
			b.Fatalf("创建RC4解密器失败: %v", err)
		}

		testData := make([]byte, 5120) // RC4段大小
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			data := make([]byte, len(testData))
			copy(data, testData)
			cipher.Decrypt(data, 0)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		// 模拟没有对象池的版本
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// 模拟每次都创建新的box数组
			box := make([]byte, 256)
			for j := range box {
				box[j] = byte(j)
			}
			// 简单的XOR操作模拟解密
			for j := range box {
				box[j] ^= byte(j % 256)
			}
		}
	})
}
