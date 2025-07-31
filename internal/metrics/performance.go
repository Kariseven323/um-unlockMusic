package metrics

import (
	"sync/atomic"
	"time"
)

// PerformanceMetrics 性能指标收集器
type PerformanceMetrics struct {
	// RC4对象池指标
	RC4PoolHits   int64 // 对象池命中次数
	RC4PoolMisses int64 // 对象池未命中次数
	RC4PoolGets   int64 // 对象池获取总次数
	RC4PoolPuts   int64 // 对象池归还总次数

	// 内存映射指标
	MmapUsage     int64 // 内存映射使用次数
	MmapFallback  int64 // 内存映射失败回退次数
	MmapTotalSize int64 // 内存映射总大小（字节）

	// SIMD优化指标
	SIMDUsage     int64 // SIMD优化使用次数
	StandardUsage int64 // 标准算法使用次数
	BatchCount    int64 // 批处理总次数

	// 解密性能指标
	DecryptionCount    int64 // 解密操作总次数
	DecryptionDuration int64 // 解密总耗时（纳秒）
	TotalBytesDecrypted int64 // 解密总字节数

	// 文件处理指标
	FilesProcessed int64 // 处理文件总数
	FilesSucceeded int64 // 成功处理文件数
	FilesFailed    int64 // 失败处理文件数
}

// 全局性能指标实例
var GlobalMetrics = &PerformanceMetrics{}

// RC4Pool指标记录
func (m *PerformanceMetrics) RecordRC4PoolHit() {
	atomic.AddInt64(&m.RC4PoolHits, 1)
	atomic.AddInt64(&m.RC4PoolGets, 1)
}

func (m *PerformanceMetrics) RecordRC4PoolMiss() {
	atomic.AddInt64(&m.RC4PoolMisses, 1)
	atomic.AddInt64(&m.RC4PoolGets, 1)
}

func (m *PerformanceMetrics) RecordRC4PoolPut() {
	atomic.AddInt64(&m.RC4PoolPuts, 1)
}

// 内存映射指标记录
func (m *PerformanceMetrics) RecordMmapUsage(size int64) {
	atomic.AddInt64(&m.MmapUsage, 1)
	atomic.AddInt64(&m.MmapTotalSize, size)
}

func (m *PerformanceMetrics) RecordMmapFallback() {
	atomic.AddInt64(&m.MmapFallback, 1)
}

// SIMD指标记录
func (m *PerformanceMetrics) RecordSIMDUsage() {
	atomic.AddInt64(&m.SIMDUsage, 1)
}

func (m *PerformanceMetrics) RecordStandardUsage() {
	atomic.AddInt64(&m.StandardUsage, 1)
}

func (m *PerformanceMetrics) RecordBatch() {
	atomic.AddInt64(&m.BatchCount, 1)
}

// 解密性能指标记录
func (m *PerformanceMetrics) RecordDecryption(duration time.Duration, bytesDecrypted int64) {
	atomic.AddInt64(&m.DecryptionCount, 1)
	atomic.AddInt64(&m.DecryptionDuration, int64(duration))
	atomic.AddInt64(&m.TotalBytesDecrypted, bytesDecrypted)
}

// 文件处理指标记录
func (m *PerformanceMetrics) RecordFileProcessed() {
	atomic.AddInt64(&m.FilesProcessed, 1)
}

func (m *PerformanceMetrics) RecordFileSucceeded() {
	atomic.AddInt64(&m.FilesSucceeded, 1)
}

func (m *PerformanceMetrics) RecordFileFailed() {
	atomic.AddInt64(&m.FilesFailed, 1)
}

// GetSnapshot 获取当前指标快照
func (m *PerformanceMetrics) GetSnapshot() MetricsSnapshot {
	return MetricsSnapshot{
		RC4PoolHits:         atomic.LoadInt64(&m.RC4PoolHits),
		RC4PoolMisses:       atomic.LoadInt64(&m.RC4PoolMisses),
		RC4PoolGets:         atomic.LoadInt64(&m.RC4PoolGets),
		RC4PoolPuts:         atomic.LoadInt64(&m.RC4PoolPuts),
		MmapUsage:           atomic.LoadInt64(&m.MmapUsage),
		MmapFallback:        atomic.LoadInt64(&m.MmapFallback),
		MmapTotalSize:       atomic.LoadInt64(&m.MmapTotalSize),
		SIMDUsage:           atomic.LoadInt64(&m.SIMDUsage),
		StandardUsage:       atomic.LoadInt64(&m.StandardUsage),
		BatchCount:          atomic.LoadInt64(&m.BatchCount),
		DecryptionCount:     atomic.LoadInt64(&m.DecryptionCount),
		DecryptionDuration:  atomic.LoadInt64(&m.DecryptionDuration),
		TotalBytesDecrypted: atomic.LoadInt64(&m.TotalBytesDecrypted),
		FilesProcessed:      atomic.LoadInt64(&m.FilesProcessed),
		FilesSucceeded:      atomic.LoadInt64(&m.FilesSucceeded),
		FilesFailed:         atomic.LoadInt64(&m.FilesFailed),
	}
}

// MetricsSnapshot 指标快照
type MetricsSnapshot struct {
	RC4PoolHits         int64
	RC4PoolMisses       int64
	RC4PoolGets         int64
	RC4PoolPuts         int64
	MmapUsage           int64
	MmapFallback        int64
	MmapTotalSize       int64
	SIMDUsage           int64
	StandardUsage       int64
	BatchCount          int64
	DecryptionCount     int64
	DecryptionDuration  int64
	TotalBytesDecrypted int64
	FilesProcessed      int64
	FilesSucceeded      int64
	FilesFailed         int64
}

// GetRC4PoolHitRate 获取RC4对象池命中率
func (s *MetricsSnapshot) GetRC4PoolHitRate() float64 {
	if s.RC4PoolGets == 0 {
		return 0
	}
	return float64(s.RC4PoolHits) / float64(s.RC4PoolGets)
}

// GetMmapSuccessRate 获取内存映射成功率
func (s *MetricsSnapshot) GetMmapSuccessRate() float64 {
	total := s.MmapUsage + s.MmapFallback
	if total == 0 {
		return 0
	}
	return float64(s.MmapUsage) / float64(total)
}

// GetSIMDUsageRate 获取SIMD使用率
func (s *MetricsSnapshot) GetSIMDUsageRate() float64 {
	total := s.SIMDUsage + s.StandardUsage
	if total == 0 {
		return 0
	}
	return float64(s.SIMDUsage) / float64(total)
}

// GetAverageDecryptionSpeed 获取平均解密速度（字节/秒）
func (s *MetricsSnapshot) GetAverageDecryptionSpeed() float64 {
	if s.DecryptionDuration == 0 {
		return 0
	}
	seconds := float64(s.DecryptionDuration) / float64(time.Second)
	return float64(s.TotalBytesDecrypted) / seconds
}

// GetFileSuccessRate 获取文件处理成功率
func (s *MetricsSnapshot) GetFileSuccessRate() float64 {
	if s.FilesProcessed == 0 {
		return 0
	}
	return float64(s.FilesSucceeded) / float64(s.FilesProcessed)
}

// Reset 重置所有指标
func (m *PerformanceMetrics) Reset() {
	atomic.StoreInt64(&m.RC4PoolHits, 0)
	atomic.StoreInt64(&m.RC4PoolMisses, 0)
	atomic.StoreInt64(&m.RC4PoolGets, 0)
	atomic.StoreInt64(&m.RC4PoolPuts, 0)
	atomic.StoreInt64(&m.MmapUsage, 0)
	atomic.StoreInt64(&m.MmapFallback, 0)
	atomic.StoreInt64(&m.MmapTotalSize, 0)
	atomic.StoreInt64(&m.SIMDUsage, 0)
	atomic.StoreInt64(&m.StandardUsage, 0)
	atomic.StoreInt64(&m.BatchCount, 0)
	atomic.StoreInt64(&m.DecryptionCount, 0)
	atomic.StoreInt64(&m.DecryptionDuration, 0)
	atomic.StoreInt64(&m.TotalBytesDecrypted, 0)
	atomic.StoreInt64(&m.FilesProcessed, 0)
	atomic.StoreInt64(&m.FilesSucceeded, 0)
	atomic.StoreInt64(&m.FilesFailed, 0)
}
