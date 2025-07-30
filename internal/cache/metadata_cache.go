package cache

import (
	"crypto/md5"
	"fmt"
	"sync"
	"time"

	"unlock-music.dev/cli/algo/common"
)

// MetadataEntry 元数据缓存条目
type MetadataEntry struct {
	Meta      common.AudioMeta
	CoverData []byte
	Timestamp time.Time
	FileHash  string
}

// MetadataCache 元数据缓存
type MetadataCache struct {
	cache    map[string]*MetadataEntry
	mutex    sync.RWMutex
	maxSize  int
	ttl      time.Duration
}

var (
	globalMetadataCache *MetadataCache
	cacheOnce           sync.Once
)

// GetGlobalMetadataCache 获取全局元数据缓存实例
func GetGlobalMetadataCache() *MetadataCache {
	cacheOnce.Do(func() {
		globalMetadataCache = NewMetadataCache(1000, 30*time.Minute)
	})
	return globalMetadataCache
}

// NewMetadataCache 创建新的元数据缓存
func NewMetadataCache(maxSize int, ttl time.Duration) *MetadataCache {
	cache := &MetadataCache{
		cache:   make(map[string]*MetadataEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
	
	// 启动清理goroutine
	go cache.cleanupExpired()
	
	return cache
}

// generateKey 生成缓存键
func (mc *MetadataCache) generateKey(filePath string, fileSize int64, modTime time.Time) string {
	data := fmt.Sprintf("%s:%d:%d", filePath, fileSize, modTime.Unix())
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// Get 获取缓存的元数据
func (mc *MetadataCache) Get(filePath string, fileSize int64, modTime time.Time) (*MetadataEntry, bool) {
	key := mc.generateKey(filePath, fileSize, modTime)
	
	mc.mutex.RLock()
	entry, exists := mc.cache[key]
	mc.mutex.RUnlock()
	
	if !exists {
		return nil, false
	}
	
	// 检查是否过期
	if time.Since(entry.Timestamp) > mc.ttl {
		mc.Delete(key)
		return nil, false
	}
	
	return entry, true
}

// Put 存储元数据到缓存
func (mc *MetadataCache) Put(filePath string, fileSize int64, modTime time.Time, meta common.AudioMeta, coverData []byte) {
	key := mc.generateKey(filePath, fileSize, modTime)
	
	entry := &MetadataEntry{
		Meta:      meta,
		CoverData: coverData,
		Timestamp: time.Now(),
		FileHash:  key,
	}
	
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	// 如果缓存已满，删除最旧的条目
	if len(mc.cache) >= mc.maxSize {
		mc.evictOldest()
	}
	
	mc.cache[key] = entry
}

// Delete 删除缓存条目
func (mc *MetadataCache) Delete(key string) {
	mc.mutex.Lock()
	delete(mc.cache, key)
	mc.mutex.Unlock()
}

// evictOldest 删除最旧的缓存条目
func (mc *MetadataCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, entry := range mc.cache {
		if oldestKey == "" || entry.Timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.Timestamp
		}
	}
	
	if oldestKey != "" {
		delete(mc.cache, oldestKey)
	}
}

// cleanupExpired 清理过期的缓存条目
func (mc *MetadataCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟清理一次
	defer ticker.Stop()
	
	for range ticker.C {
		mc.mutex.Lock()
		now := time.Now()
		
		for key, entry := range mc.cache {
			if now.Sub(entry.Timestamp) > mc.ttl {
				delete(mc.cache, key)
			}
		}
		
		mc.mutex.Unlock()
	}
}

// GetStats 获取缓存统计信息
func (mc *MetadataCache) GetStats() map[string]interface{} {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	
	return map[string]interface{}{
		"size":     len(mc.cache),
		"max_size": mc.maxSize,
		"ttl":      mc.ttl.String(),
	}
}

// Clear 清空缓存
func (mc *MetadataCache) Clear() {
	mc.mutex.Lock()
	mc.cache = make(map[string]*MetadataEntry)
	mc.mutex.Unlock()
}

// 便利函数

// GetMetadata 获取缓存的元数据
func GetMetadata(filePath string, fileSize int64, modTime time.Time) (*MetadataEntry, bool) {
	return GetGlobalMetadataCache().Get(filePath, fileSize, modTime)
}

// PutMetadata 存储元数据到缓存
func PutMetadata(filePath string, fileSize int64, modTime time.Time, meta common.AudioMeta, coverData []byte) {
	GetGlobalMetadataCache().Put(filePath, fileSize, modTime, meta, coverData)
}

// ClearMetadataCache 清空元数据缓存
func ClearMetadataCache() {
	GetGlobalMetadataCache().Clear()
}
