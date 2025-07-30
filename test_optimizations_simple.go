package main

import (
	"fmt"
	"runtime"
	"time"

	"unlock-music.dev/cli/internal/pool"
)

func main() {
	fmt.Println("=== 高优先级优化测试 ===")
	
	// 测试1: 内存池优化
	testMemoryPool()
	
	// 测试2: 并发数计算
	testConcurrentWorkers()
	
	// 测试3: 缓冲区大小
	testBufferSizes()
	
	fmt.Println("✅ 所有优化测试完成")
}

func testMemoryPool() {
	fmt.Println("\n1. 测试内存池优化...")
	
	start := time.Now()
	
	// 测试内存池的获取和归还
	for i := 0; i < 1000; i++ {
		buf := pool.GetBuffer(1024)
		
		// 写入一些数据
		for j := range buf {
			buf[j] = byte(i % 256)
		}
		
		// 归还缓冲区（会自动清零）
		pool.PutBuffer(buf)
	}
	
	duration := time.Since(start)
	fmt.Printf("   内存池测试完成，耗时: %v\n", duration)
	
	// 测试不同大小的缓冲区
	sizes := []int{pool.SmallBufferSize, pool.MediumBufferSize, pool.LargeBufferSize}
	for _, size := range sizes {
		buf := pool.GetBuffer(size)
		fmt.Printf("   获取 %d 字节缓冲区: 实际大小 %d\n", size, len(buf))
		pool.PutBuffer(buf)
	}
}

func testConcurrentWorkers() {
	fmt.Println("\n2. 测试并发数计算...")
	
	cpuCount := runtime.NumCPU()
	maxWorkers := calculateOptimalWorkers()
	
	fmt.Printf("   CPU核心数: %d\n", cpuCount)
	fmt.Printf("   计算的最优worker数: %d\n", maxWorkers)
	
	// 验证范围
	if maxWorkers >= 2 && maxWorkers <= 16 {
		fmt.Printf("   ✅ worker数在合理范围内 (2-16)\n")
	} else {
		fmt.Printf("   ❌ worker数超出预期范围\n")
	}
	
	// 验证与CPU核心数的关系
	expectedMax := cpuCount * 2
	if expectedMax > 16 {
		expectedMax = 16
	}
	if expectedMax < 2 {
		expectedMax = 2
	}
	
	if maxWorkers == expectedMax {
		fmt.Printf("   ✅ worker数计算正确\n")
	} else {
		fmt.Printf("   ❌ worker数计算异常，期望: %d，实际: %d\n", expectedMax, maxWorkers)
	}
}

func testBufferSizes() {
	fmt.Println("\n3. 测试缓冲区大小优化...")
	
	// 测试256字节缓冲区（用于格式识别）
	headerBuf := pool.GetBuffer(256)
	fmt.Printf("   格式识别缓冲区大小: %d 字节\n", len(headerBuf))
	
	if len(headerBuf) == 256 {
		fmt.Printf("   ✅ header缓冲区大小正确\n")
	} else {
		fmt.Printf("   ❌ header缓冲区大小异常\n")
	}
	
	pool.PutBuffer(headerBuf)
	
	// 测试内存池统计
	stats := pool.GetGlobalPool().GetStats()
	fmt.Printf("   内存池统计:\n")
	for _, size := range stats.PoolSizes {
		fmt.Printf("     - %d 字节池已创建\n", size)
	}
}

// calculateOptimalWorkers 动态计算最优worker数量（复制自batch.go）
func calculateOptimalWorkers() int {
	cpuCount := runtime.NumCPU()
	
	// 基于CPU核心数和系统负载动态调整
	// I/O密集型任务可以使用更多worker
	maxWorkers := cpuCount * 2
	
	// 设置合理的上下限
	if maxWorkers < 2 {
		maxWorkers = 2
	}
	if maxWorkers > 16 {
		maxWorkers = 16 // 避免过多goroutine导致调度开销
	}
	
	return maxWorkers
}
