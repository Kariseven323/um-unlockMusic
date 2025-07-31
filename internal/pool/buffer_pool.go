package pool

import (
	"fmt"
	"sync"
)

// BufferPool 缓冲区池，用于复用不同大小的缓冲区
type BufferPool struct {
	pools map[int]*sync.Pool
	mutex sync.RWMutex
}

// 预定义的缓冲区大小
const (
	SmallBufferSize  = 4 * 1024        // 4KB - 用于小文件或header读取
	MediumBufferSize = 64 * 1024       // 64KB - 用于一般文件处理
	LargeBufferSize  = 1024 * 1024     // 1MB - 用于大文件处理
	XLargeBufferSize = 4 * 1024 * 1024 // 4MB - 用于I/O密集型操作
)

// 全局缓冲区池实例
var (
	globalBufferPool *BufferPool
	once             sync.Once
)

// GetGlobalPool 获取全局缓冲区池实例
func GetGlobalPool() *BufferPool {
	once.Do(func() {
		globalBufferPool = NewBufferPool()
	})
	return globalBufferPool
}

// NewBufferPool 创建新的缓冲区池
func NewBufferPool() *BufferPool {
	bp := &BufferPool{
		pools: make(map[int]*sync.Pool),
	}

	// 预创建常用大小的池
	bp.initPool(SmallBufferSize)
	bp.initPool(MediumBufferSize)
	bp.initPool(LargeBufferSize)
	bp.initPool(XLargeBufferSize)

	return bp
}

// initPool 初始化指定大小的缓冲区池
func (bp *BufferPool) initPool(size int) {
	bp.pools[size] = &sync.Pool{
		New: func() any {
			return make([]byte, size)
		},
	}
}

// Get 获取指定大小的缓冲区
func (bp *BufferPool) Get(size int) []byte {
	// 找到最接近且不小于请求大小的池
	poolSize := bp.findBestPoolSize(size)

	bp.mutex.RLock()
	pool, exists := bp.pools[poolSize]
	bp.mutex.RUnlock()

	if !exists {
		// 如果没有对应大小的池，创建一个
		bp.mutex.Lock()
		// 双重检查
		if pool, exists = bp.pools[poolSize]; !exists {
			bp.initPool(poolSize)
			pool = bp.pools[poolSize]
		}
		bp.mutex.Unlock()
	}

	buf := pool.Get().([]byte)

	// 如果缓冲区大小不匹配，重新分配
	if len(buf) != poolSize {
		buf = make([]byte, poolSize)
	}

	// 返回请求大小的切片
	return buf[:size]
}

// Put 归还缓冲区到池中
func (bp *BufferPool) Put(buf []byte) {
	if len(buf) == 0 {
		return
	}

	// 获取原始容量
	capacity := cap(buf)
	poolSize := bp.findBestPoolSize(capacity)

	bp.mutex.RLock()
	pool, exists := bp.pools[poolSize]
	bp.mutex.RUnlock()

	if exists && capacity == poolSize {
		// 重置切片长度为容量
		buf = buf[:capacity]
		// 优化：只对敏感数据进行清零，提升性能
		// 对于音频数据缓冲区，通常不需要完全清零
		if capacity <= SmallBufferSize {
			// 小缓冲区可能包含敏感header信息，需要清零
			for i := range buf {
				buf[i] = 0
			}
		} else {
			// 大缓冲区主要用于音频数据，只清零前64字节（可能的header）
			clearSize := 64
			if len(buf) < clearSize {
				clearSize = len(buf)
			}
			for i := 0; i < clearSize; i++ {
				buf[i] = 0
			}
		}
		pool.Put(buf)
	}
	// 如果大小不匹配，直接丢弃，让GC回收
}

// findBestPoolSize 找到最适合的池大小
func (bp *BufferPool) findBestPoolSize(size int) int {
	// 对于小于等于预定义大小的，直接使用预定义大小
	if size <= SmallBufferSize {
		return SmallBufferSize
	}
	if size <= MediumBufferSize {
		return MediumBufferSize
	}
	if size <= LargeBufferSize {
		return LargeBufferSize
	}
	if size <= XLargeBufferSize {
		return XLargeBufferSize
	}

	// 对于更大的尺寸，向上取整到最近的2的幂次
	return nextPowerOfTwo(size)
}

// nextPowerOfTwo 计算大于等于n的最小2的幂次
func nextPowerOfTwo(n int) int {
	if n <= 0 {
		return 1
	}

	// 如果已经是2的幂次，直接返回
	if n&(n-1) == 0 {
		return n
	}

	// 找到最高位
	power := 1
	for power < n {
		power <<= 1
	}

	return power
}

// Stats 缓冲区池统计信息
type Stats struct {
	PoolSizes []int            `json:"pool_sizes"`
	PoolStats map[int]PoolStat `json:"pool_stats"`
}

// PoolStat 单个池的统计信息
type PoolStat struct {
	Size  int `json:"size"`
	InUse int `json:"in_use"` // 当前使用中的缓冲区数量（估算）
	Total int `json:"total"`  // 总分配的缓冲区数量（估算）
}

// GetStats 获取缓冲区池统计信息
func (bp *BufferPool) GetStats() Stats {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()

	stats := Stats{
		PoolSizes: make([]int, 0, len(bp.pools)),
		PoolStats: make(map[int]PoolStat),
	}

	for size := range bp.pools {
		stats.PoolSizes = append(stats.PoolSizes, size)
		// 注意：sync.Pool 不提供准确的统计信息
		// 这里只是提供一个基本的结构
		stats.PoolStats[size] = PoolStat{
			Size:  size,
			InUse: 0, // sync.Pool 无法准确统计
			Total: 0, // sync.Pool 无法准确统计
		}
	}

	return stats
}

// 便利函数

// GetSmallBuffer 获取小缓冲区 (4KB)
func GetSmallBuffer() []byte {
	return GetGlobalPool().Get(SmallBufferSize)
}

// GetMediumBuffer 获取中等缓冲区 (64KB)
func GetMediumBuffer() []byte {
	return GetGlobalPool().Get(MediumBufferSize)
}

// GetLargeBuffer 获取大缓冲区 (1MB)
func GetLargeBuffer() []byte {
	return GetGlobalPool().Get(LargeBufferSize)
}

// GetXLargeBuffer 获取超大缓冲区 (4MB) - 用于I/O密集型操作
func GetXLargeBuffer() []byte {
	return GetGlobalPool().Get(XLargeBufferSize)
}

// GetBuffer 获取指定大小的缓冲区
func GetBuffer(size int) []byte {
	return GetGlobalPool().Get(size)
}

// PutBuffer 归还缓冲区
func PutBuffer(buf []byte) {
	GetGlobalPool().Put(buf)
}

// 缓冲区大小选择缓存
var bufferSizeCache = make(map[string]int)
var bufferSizeCacheMutex sync.RWMutex

// GetOptimalBufferSize 根据文件大小和类型获取最优缓冲区大小
func GetOptimalBufferSize(fileSize int64, fileExt string) int {
	// 生成缓存键
	cacheKey := fmt.Sprintf("%d_%s", fileSize/(1024*1024), fileExt) // 以MB为单位缓存

	// 检查缓存
	bufferSizeCacheMutex.RLock()
	if cachedSize, exists := bufferSizeCache[cacheKey]; exists {
		bufferSizeCacheMutex.RUnlock()
		return cachedSize
	}
	bufferSizeCacheMutex.RUnlock()

	// 基于文件大小的基础选择
	var baseSize int

	switch {
	case fileSize < 1024*1024: // <1MB
		baseSize = SmallBufferSize // 4KB
	case fileSize < 10*1024*1024: // <10MB
		baseSize = MediumBufferSize // 64KB
	case fileSize < 100*1024*1024: // <100MB
		baseSize = LargeBufferSize // 1MB
	default: // >=100MB
		baseSize = XLargeBufferSize // 4MB
	}

	// 基于文件类型的调整
	switch fileExt {
	case ".ncm", ".qmc", ".qmcflac", ".qmcogg":
		// 加密音频文件，需要更多I/O缓冲
		if baseSize < MediumBufferSize {
			baseSize = MediumBufferSize
		}
	case ".kgm", ".vpr":
		// 这些格式通常文件较大
		if baseSize < LargeBufferSize {
			baseSize = LargeBufferSize
		}
	case ".tm0", ".tm2", ".tm3", ".tm6":
		// TM格式通常较小，使用小缓冲区即可
		if baseSize > MediumBufferSize {
			baseSize = MediumBufferSize
		}
	}

	// 缓存结果
	bufferSizeCacheMutex.Lock()
	bufferSizeCache[cacheKey] = baseSize
	bufferSizeCacheMutex.Unlock()

	return baseSize
}

// GetOptimalBuffer 获取最优大小的缓冲区
func GetOptimalBuffer(fileSize int64, fileExt string) []byte {
	size := GetOptimalBufferSize(fileSize, fileExt)
	return GetGlobalPool().Get(size)
}
