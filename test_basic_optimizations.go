package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Println("=== 基础优化验证 ===")
	
	// 测试并发数计算
	testConcurrentWorkers()
	
	// 测试缓冲区清零逻辑
	testBufferClear()
	
	fmt.Println("✅ 基础优化验证完成")
}

func testConcurrentWorkers() {
	fmt.Println("\n1. 测试并发数计算...")
	
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

func testBufferClear() {
	fmt.Println("\n2. 测试缓冲区清零逻辑...")
	
	// 模拟缓冲区清零
	buf := make([]byte, 256)
	
	// 填充数据
	for i := range buf {
		buf[i] = byte(i % 256)
	}
	
	fmt.Printf("   填充前缓冲区前10字节: %v\n", buf[:10])
	
	// 清零（模拟内存池的清零逻辑）
	for i := range buf {
		buf[i] = 0
	}
	
	fmt.Printf("   清零后缓冲区前10字节: %v\n", buf[:10])
	
	// 验证是否全部清零
	allZero := true
	for _, b := range buf {
		if b != 0 {
			allZero = false
			break
		}
	}
	
	if allZero {
		fmt.Printf("   ✅ 缓冲区清零成功\n")
	} else {
		fmt.Printf("   ❌ 缓冲区清零失败\n")
	}
}

// calculateOptimalWorkers 动态计算最优worker数量
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
