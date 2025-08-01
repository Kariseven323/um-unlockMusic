# mgg文件解密问题修复任务

## 任务概述
修复mgg文件解密的两个核心问题：
1. 前端mgg文件选中后解密失败
2. 后端直接解密mgg文件输出扩展名错误（应该是.ogg但输出成.mp3）

## 问题分析

### 根本原因
1. **后端输出扩展名问题**：
   - 后端使用格式嗅探确定输出扩展名：`sniff.AudioExtensionWithFallback(header.Bytes(), ".mp3")`
   - 当格式嗅探失败时，fallback到`.mp3`
   - mgg文件解密后应该是ogg格式，但嗅探失败导致错误的mp3扩展名

2. **前端输出格式推测缺失**：
   - `music_unlock_gui/core/processor.py`中`get_output_filename`方法缺少mgg格式处理
   - 没有mgg→ogg的映射，导致前端预期与实际输出不符

## 解决方案实施

### 步骤1：修正前端输出格式推测 ✅
**文件**: `music_unlock_gui/core/processor.py`
**修改**: 在`get_output_filename`方法中添加mgg格式处理
**内容**:
```python
elif input_ext.startswith('.mflac'):
    # mflac及其变体（mflac0, mflac1, mflaca等）输出为flac
    return f"{base_name}.flac"
elif input_ext.startswith('.mgg'):
    # mgg及其变体（mgg0, mgg1, mgga等）输出为ogg
    return f"{base_name}.ogg"
```

### 步骤2：增强后端格式嗅探fallback逻辑 ✅
**文件**: `internal/sniff/audio.go`
**修改**: 添加智能fallback函数
**内容**:
```go
// AudioExtensionWithSmartFallback 使用基于输入文件扩展名的智能fallback
func AudioExtensionWithSmartFallback(header []byte, inputExt string) string

// getSmartFallback 根据输入文件扩展名返回预期输出格式
func getSmartFallback(inputExt string) string
```

**文件**: `cmd/um/main.go`
**修改**: 使用智能fallback替代固定fallback
**内容**:
```go
// 使用智能fallback，根据输入文件扩展名推测输出格式
inputExt := filepath.Ext(inputFile)
params.AudioExt = sniff.AudioExtensionWithSmartFallback(header.Bytes(), inputExt)
```

## 预期效果
1. **前端改进**：
   - mgg文件在文件列表中正确显示预期输出为.ogg格式
   - 解决前端mgg文件解密失败的问题

2. **后端改进**：
   - mgg文件解密后输出正确的.ogg扩展名
   - 支持mgg变体格式（mgg0, mgg1等）
   - 同时修正mflac格式的处理

## 测试验证
- [x] 测试前端添加mgg文件，确认预期输出格式显示为.ogg ✅
- [x] 测试后端智能fallback功能，确认mgg→ogg映射正确 ✅
- [x] 测试mgg变体格式（mgg0, mgg1等）正确映射 ✅
- [x] 验证mflac格式也正确输出为.flac ✅
- [x] 测试实际mgg文件解密，确认输出正确的.ogg扩展名 ✅

### 测试结果
**前端测试**：
```
test.mgg     -> test.ogg
test.mgg0    -> test.ogg
test.mgg1    -> test.ogg
test.mflac   -> test.flac
test.mflac0  -> test.flac
test.qmcogg  -> test.ogg
```

**后端智能fallback测试**：
```
✓ .mgg -> .ogg (期望: .ogg)
✓ .mgg0 -> .ogg (期望: .ogg)
✓ .mgg1 -> .ogg (期望: .ogg)
✓ .mgga -> .ogg (期望: .ogg)
✓ .mflac -> .flac (期望: .flac)
✓ .mflac0 -> .flac (期望: .flac)
✓ .mflac1 -> .flac (期望: .flac)
✓ .qmcflac -> .flac (期望: .flac)
✓ .qmcogg -> .ogg (期望: .ogg)
```

**实际mgg文件测试**：
```
文件: 身后 - 林俊杰.mgg
输入扩展名: .mgg
检测扩展名: .ogg
输出文件: 身后 - 林俊杰.ogg
文件内容: Ogg data, Vorbis audio, stereo, 44100 Hz, ~320000 bps
前端支持: True
前端预期输出: 身后 - 林俊杰.ogg
```

## 问题解决状态
✅ **已完全解决**

### 根本原因发现
问题的根本原因是**格式嗅探器的优先级问题**：
- mgg文件解密后确实是Ogg格式（以"OggS"开头）
- 但是mp3嗅探器在Ogg数据中误识别了MP3帧头
- 由于Go map遍历顺序随机，有时mp3嗅探器会在ogg嗅探器之前执行
- 导致Ogg格式被错误识别为MP3格式

### 最终解决方案
1. **前端问题**：添加了mgg→ogg的输出格式推测
2. **后端问题**：
   - 实现了智能fallback机制（作为备用方案）
   - **关键修复**：重构了格式嗅探逻辑，按优先级顺序检查格式
   - 确保具有明确魔术头的格式（如Ogg、FLAC）优先于可能误判的格式（如MP3）

### 修复效果
- **拖拽模式**：mgg文件正确输出.ogg扩展名 ✅
- **命令行模式**：mgg文件正确输出.ogg扩展名 ✅
- **前端界面**：正确识别并显示预期输出格式 ✅

## 相关文件
- `music_unlock_gui/core/processor.py` - 前端输出格式推测
- `internal/sniff/audio.go` - 格式嗅探和智能fallback
- `cmd/um/main.go` - 后端主处理逻辑
