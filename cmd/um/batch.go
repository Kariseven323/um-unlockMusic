package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"unlock-music.dev/cli/algo/common"
	"unlock-music.dev/cli/internal/ffmpeg"
	"unlock-music.dev/cli/internal/sniff"
	"unlock-music.dev/cli/internal/utils"
)

// 批处理相关常量
const (
	metadataTimeout = 10 * time.Second // 元数据获取超时时间
	writeTimeout    = time.Minute      // 写入操作超时时间
)

// BatchRequest 批处理请求结构
type BatchRequest struct {
	Files   []FileTask     `json:"files"`
	Options ProcessOptions `json:"options"`
}

// FileTask 单个文件处理任务
type FileTask struct {
	InputPath  string `json:"input_path"`
	OutputPath string `json:"output_path,omitempty"`
	Priority   int    `json:"priority,omitempty"`  // 任务优先级，数值越小优先级越高
	FileSize   int64  `json:"file_size,omitempty"` // 文件大小，用于优先级计算
}

// ProcessOptions 处理选项
type ProcessOptions struct {
	RemoveSource    bool   `json:"remove_source,omitempty"`
	UpdateMetadata  bool   `json:"update_metadata,omitempty"`
	OverwriteOutput bool   `json:"overwrite_output,omitempty"`
	SkipNoop        bool   `json:"skip_noop,omitempty"`
	KggDbPath       string `json:"kgg_db_path,omitempty"`
	NamingFormat    string `json:"naming_format,omitempty"`
}

// ProcessResult 处理结果
type ProcessResult struct {
	InputPath   string `json:"input_path"`
	OutputPath  string `json:"output_path,omitempty"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
	ProcessTime int64  `json:"process_time_ms"`
}

// BatchResponse 批处理响应
type BatchResponse struct {
	Results      []ProcessResult `json:"results"`
	TotalFiles   int             `json:"total_files"`
	SuccessCount int             `json:"success_count"`
	FailedCount  int             `json:"failed_count"`
	TotalTime    int64           `json:"total_time_ms"`
}

// taskWithIndex 带索引的任务
type taskWithIndex struct {
	index int
	task  FileTask
}

// resultWithIndex 带索引的结果
type resultWithIndex struct {
	index  int
	result ProcessResult
}

// pipelineData 流水线数据结构
type pipelineData struct {
	index int
	task  FileTask

	// 解密相关数据
	audioExt      string
	audioData     []byte // 解密后的音频数据
	tempAudioFile string // 临时音频文件路径，用于资源清理

	// 元数据相关
	metadata    common.AudioMeta
	albumArt    []byte
	albumArtExt string
	coverGetter common.CoverImageGetter // 延迟加载封面图片

	// 输出相关
	outputPath string

	// 错误处理
	error error
}

// batchProcessor 批处理器
type batchProcessor struct {
	logger  *zap.Logger
	options ProcessOptions

	// 复用的processor实例，避免重复初始化
	baseProcessor *processor

	// 并发控制
	maxWorkers int

	// 流水线并发控制
	enablePipeline bool
	pipelineStages int
}

// newBatchProcessor 创建批处理器
func newBatchProcessor(options ProcessOptions, logger *zap.Logger) *batchProcessor {
	baseProc := &processor{
		logger:          logger,
		skipNoopDecoder: options.SkipNoop,
		removeSource:    options.RemoveSource,
		updateMetadata:  options.UpdateMetadata,
		overwriteOutput: options.OverwriteOutput,
		kggDbPath:       options.KggDbPath,
		namingFormat:    options.NamingFormat,
	}

	// 动态设置并发数
	maxWorkers := calculateOptimalWorkers()

	return &batchProcessor{
		logger:         logger,
		options:        options,
		baseProcessor:  baseProc,
		maxWorkers:     maxWorkers,
		enablePipeline: true, // 启用流水线并发
		pipelineStages: 2,    // 2个阶段：解密和写入
	}
}

// processBatch 处理批量任务（并发版本）
func (bp *batchProcessor) processBatch(request *BatchRequest) *BatchResponse {
	// 根据实际文件信息动态调整worker数量
	bp.optimizeWorkerCount(request.Files)

	// 如果启用流水线并发且文件数量足够多，使用流水线模式
	if bp.enablePipeline && len(request.Files) >= 4 {
		return bp.processBatchPipeline(request)
	}

	// 否则使用传统并发模式
	startTime := time.Now()

	response := &BatchResponse{
		Results:    make([]ProcessResult, len(request.Files)),
		TotalFiles: len(request.Files),
	}

	// 计算任务优先级并排序
	bp.calculateTaskPriorities(request.Files)
	bp.sortTasksByPriority(request.Files)

	bp.logger.Info("开始并发批处理",
		zap.Int("文件数量", len(request.Files)),
		zap.Int("并发数", bp.maxWorkers))

	// 创建工作通道
	taskChan := make(chan taskWithIndex, len(request.Files))
	resultChan := make(chan resultWithIndex, len(request.Files))

	// 启动工作协程
	var wg sync.WaitGroup
	for i := 0; i < bp.maxWorkers; i++ {
		wg.Add(1)
		go bp.worker(&wg, taskChan, resultChan)
	}

	// 发送任务
	for i, task := range request.Files {
		taskChan <- taskWithIndex{index: i, task: task}
	}
	close(taskChan)

	// 等待所有工作完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	for result := range resultChan {
		response.Results[result.index] = result.result
		if result.result.Success {
			response.SuccessCount++
		} else {
			response.FailedCount++
		}
	}

	response.TotalTime = time.Since(startTime).Milliseconds()

	bp.logger.Info("并发批处理完成",
		zap.Int("成功", response.SuccessCount),
		zap.Int("失败", response.FailedCount),
		zap.Int64("总耗时(ms)", response.TotalTime))

	return response
}

// processFileTask 处理单个文件任务
func (bp *batchProcessor) processFileTask(task FileTask) ProcessResult {
	startTime := time.Now()

	result := ProcessResult{
		InputPath: task.InputPath,
	}

	// 检查输入文件是否存在
	if _, err := os.Stat(task.InputPath); os.IsNotExist(err) {
		result.Error = fmt.Sprintf("输入文件不存在: %s", task.InputPath)
		result.ProcessTime = time.Since(startTime).Milliseconds()
		return result
	}

	// 确定输出路径
	outputPath := task.OutputPath
	if outputPath == "" {
		// 如果没有指定输出路径，使用输入文件的目录
		outputPath = filepath.Dir(task.InputPath)
	}

	// 设置processor的输出目录
	bp.baseProcessor.outputDir = outputPath
	bp.baseProcessor.inputDir = filepath.Dir(task.InputPath)

	// 处理文件
	err := bp.baseProcessor.processFile(task.InputPath)
	if err != nil {
		result.Error = err.Error()
		bp.logger.Error("处理文件失败",
			zap.String("文件", task.InputPath),
			zap.Error(err))
	} else {
		result.Success = true
		result.OutputPath = outputPath
		bp.logger.Debug("文件处理成功", zap.String("文件", task.InputPath))
	}

	result.ProcessTime = time.Since(startTime).Milliseconds()
	return result
}

// readBatchRequest 从stdin读取批处理请求
func readBatchRequest() (*BatchRequest, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("读取stdin失败: %w", err)
	}

	var request BatchRequest
	if err := json.Unmarshal(data, &request); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	// 设置默认命名格式
	if request.Options.NamingFormat == "" {
		request.Options.NamingFormat = "auto"
	}

	return &request, nil
}

// writeBatchResponse 输出批处理响应
func writeBatchResponse(response *BatchResponse) error {
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化响应失败: %w", err)
	}

	_, err = os.Stdout.Write(data)
	if err != nil {
		return fmt.Errorf("写入响应失败: %w", err)
	}

	return nil
}

// runBatchMode 运行批处理模式
func runBatchMode(logger *zap.Logger) error {
	logger.Info("启动批处理模式")

	// 读取批处理请求
	request, err := readBatchRequest()
	if err != nil {
		return fmt.Errorf("读取批处理请求失败: %w", err)
	}

	// 创建批处理器
	batchProc := newBatchProcessor(request.Options, logger)

	// 处理批量任务
	response := batchProc.processBatch(request)

	// 输出结果
	if err := writeBatchResponse(response); err != nil {
		return fmt.Errorf("输出批处理响应失败: %w", err)
	}

	return nil
}

// worker 工作协程
func (bp *batchProcessor) worker(wg *sync.WaitGroup, taskChan <-chan taskWithIndex, resultChan chan<- resultWithIndex) {
	defer wg.Done()

	for taskWithIdx := range taskChan {
		result := bp.processFileTask(taskWithIdx.task)
		resultChan <- resultWithIndex{
			index:  taskWithIdx.index,
			result: result,
		}
	}
}

// calculateOptimalWorkers 动态计算最优worker数量
func calculateOptimalWorkers() int {
	return calculateOptimalWorkersForFiles(0, 0)
}

// calculateOptimalWorkersForFiles 基于文件数量和大小计算最优worker数量
func calculateOptimalWorkersForFiles(fileCount int, avgFileSize int64) int {
	cpuCount := runtime.NumCPU()

	// 基础worker数量：CPU核心数
	baseWorkers := cpuCount

	// 根据文件特征调整
	if fileCount > 0 {
		// 文件数量因子：更多文件可以使用更多worker
		if fileCount >= 100 {
			baseWorkers = cpuCount * 3 // 大批量文件
		} else if fileCount >= 20 {
			baseWorkers = cpuCount * 2 // 中等批量文件
		} else if fileCount <= 5 {
			baseWorkers = cpuCount / 2 // 少量文件，减少开销
			if baseWorkers < 1 {
				baseWorkers = 1
			}
		}

		// 文件大小因子：大文件更适合I/O并发
		if avgFileSize > 50*1024*1024 { // >50MB
			baseWorkers = int(float64(baseWorkers) * 1.5) // 大文件增加I/O并发
		} else if avgFileSize < 1024*1024 { // <1MB
			baseWorkers = int(float64(baseWorkers) * 0.8) // 小文件减少开销
		}
	} else {
		// 默认情况：I/O密集型任务可以使用更多worker
		baseWorkers = cpuCount * 2
	}

	// 设置合理的上下限
	if baseWorkers < 1 {
		baseWorkers = 1
	}
	if baseWorkers > 20 {
		baseWorkers = 20 // 避免过多goroutine导致调度开销
	}

	return baseWorkers
}

// optimizeWorkerCount 根据文件信息优化worker数量
func (bp *batchProcessor) optimizeWorkerCount(files []FileTask) {
	if len(files) == 0 {
		return
	}

	// 计算平均文件大小
	var totalSize int64
	validFiles := 0

	for _, file := range files {
		if stat, err := os.Stat(file.InputPath); err == nil {
			totalSize += stat.Size()
			validFiles++
		}
	}

	var avgFileSize int64
	if validFiles > 0 {
		avgFileSize = totalSize / int64(validFiles)
	}

	// 重新计算最优worker数量
	newWorkerCount := calculateOptimalWorkersForFiles(len(files), avgFileSize)

	// 更新worker数量
	if newWorkerCount != bp.maxWorkers {
		bp.logger.Info("动态调整worker数量",
			zap.Int("原worker数", bp.maxWorkers),
			zap.Int("新worker数", newWorkerCount),
			zap.Int("文件数量", len(files)),
			zap.Int64("平均文件大小", avgFileSize))
		bp.maxWorkers = newWorkerCount
	}
}

// calculateTaskPriorities 计算任务优先级
func (bp *batchProcessor) calculateTaskPriorities(tasks []FileTask) {
	for i := range tasks {
		task := &tasks[i]

		// 获取文件大小
		if task.FileSize == 0 {
			if stat, err := os.Stat(task.InputPath); err == nil {
				task.FileSize = stat.Size()
			}
		}

		// 计算优先级：小文件优先（优先级数值越小越优先）
		// 基于文件大小计算，小于1MB的文件优先级为1，其他为2
		if task.Priority == 0 {
			if task.FileSize < 1024*1024 { // 1MB
				task.Priority = 1 // 高优先级
			} else {
				task.Priority = 2 // 普通优先级
			}
		}
	}
}

// sortTasksByPriority 按优先级排序任务
func (bp *batchProcessor) sortTasksByPriority(tasks []FileTask) {
	sort.Slice(tasks, func(i, j int) bool {
		// 首先按优先级排序
		if tasks[i].Priority != tasks[j].Priority {
			return tasks[i].Priority < tasks[j].Priority
		}
		// 相同优先级时，按文件大小排序（小文件优先）
		return tasks[i].FileSize < tasks[j].FileSize
	})
}

// processBatchPipeline 流水线并发处理批量任务
func (bp *batchProcessor) processBatchPipeline(request *BatchRequest) *BatchResponse {
	startTime := time.Now()

	response := &BatchResponse{
		Results:    make([]ProcessResult, len(request.Files)),
		TotalFiles: len(request.Files),
	}

	// 计算任务优先级并排序
	bp.calculateTaskPriorities(request.Files)
	bp.sortTasksByPriority(request.Files)

	bp.logger.Info("开始流水线并发批处理",
		zap.Int("文件数量", len(request.Files)),
		zap.Int("并发数", bp.maxWorkers),
		zap.Int("流水线阶段", bp.pipelineStages))

	// 创建流水线通道
	decryptChan := make(chan taskWithIndex, bp.maxWorkers)
	writeChan := make(chan pipelineData, bp.maxWorkers)
	resultChan := make(chan resultWithIndex, len(request.Files))

	// 启动解密worker
	var decryptWg sync.WaitGroup
	for i := 0; i < bp.maxWorkers/2; i++ { // 一半worker用于解密
		decryptWg.Add(1)
		go bp.decryptWorker(&decryptWg, decryptChan, writeChan)
	}

	// 启动写入worker
	var writeWg sync.WaitGroup
	for i := 0; i < bp.maxWorkers/2; i++ { // 一半worker用于写入
		writeWg.Add(1)
		go bp.writeWorker(&writeWg, writeChan, resultChan)
	}

	// 发送解密任务
	go func() {
		for i, task := range request.Files {
			decryptChan <- taskWithIndex{index: i, task: task}
		}
		close(decryptChan)
	}()

	// 等待解密完成并关闭写入通道
	go func() {
		decryptWg.Wait()
		close(writeChan)
	}()

	// 等待写入完成并关闭结果通道
	go func() {
		writeWg.Wait()
		close(resultChan)
	}()

	// 收集结果
	for result := range resultChan {
		response.Results[result.index] = result.result
		if result.result.Success {
			response.SuccessCount++
		}
	}

	response.TotalTime = time.Since(startTime).Milliseconds()
	bp.logger.Info("流水线批处理完成",
		zap.Int("成功", response.SuccessCount),
		zap.Int("总数", response.TotalFiles),
		zap.Int64("耗时(ms)", response.TotalTime))

	return response
}

// decryptWorker 解密worker
func (bp *batchProcessor) decryptWorker(wg *sync.WaitGroup, taskChan <-chan taskWithIndex, writeChan chan<- pipelineData) {
	defer wg.Done()

	for taskWithIdx := range taskChan {
		data := pipelineData{
			index: taskWithIdx.index,
			task:  taskWithIdx.task,
		}

		bp.logger.Debug("开始解密", zap.String("文件", taskWithIdx.task.InputPath))

		// 执行解密操作
		data.error = bp.performDecryption(&data)

		writeChan <- data
	}
}

// writeWorker 写入worker
func (bp *batchProcessor) writeWorker(wg *sync.WaitGroup, writeChan <-chan pipelineData, resultChan chan<- resultWithIndex) {
	defer wg.Done()

	for data := range writeChan {
		result := ProcessResult{
			InputPath: data.task.InputPath,
		}

		if data.error != nil {
			result.Error = data.error.Error()
		} else {
			// 执行写入操作
			bp.logger.Debug("开始写入", zap.String("文件", data.task.InputPath))

			err := bp.performWrite(&data)
			if err != nil {
				result.Error = err.Error()
				bp.logger.Error("写入文件失败",
					zap.String("文件", data.task.InputPath),
					zap.Error(err))
			} else {
				result.Success = true
				result.OutputPath = data.outputPath
				bp.logger.Debug("写入文件成功",
					zap.String("输入", data.task.InputPath),
					zap.String("输出", data.outputPath))
			}
		}

		resultChan <- resultWithIndex{
			index:  data.index,
			result: result,
		}
	}
}

// performDecryption 执行解密操作
func (bp *batchProcessor) performDecryption(data *pipelineData) error {
	// 检查输入文件是否存在
	if _, err := os.Stat(data.task.InputPath); os.IsNotExist(err) {
		return fmt.Errorf("输入文件不存在: %s", data.task.InputPath)
	}

	// 打开输入文件
	file, err := os.Open(data.task.InputPath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 获取解码器
	allDec := common.GetDecoder(data.task.InputPath, bp.baseProcessor.skipNoopDecoder)
	if len(allDec) == 0 {
		return errors.New("没有找到合适的解码器")
	}

	// 查找可用的解码器
	pDec, decoderFactory, err := bp.baseProcessor.findDecoder(allDec, &common.DecoderParams{
		Reader:          file,
		Extension:       filepath.Ext(data.task.InputPath),
		FilePath:        data.task.InputPath,
		Logger:          bp.logger,
		KggDatabasePath: bp.baseProcessor.kggDbPath,
	})
	if err != nil {
		return fmt.Errorf("查找解码器失败: %w", err)
	}
	dec := *pDec

	// 读取音频头部用于格式识别
	// 修复问题：创建独立的header缓冲区，避免内存池生命周期问题
	headerBuf := make([]byte, 256)
	_, err = io.ReadFull(dec, headerBuf)
	if err != nil {
		return fmt.Errorf("读取音频头部失败: %w", err)
	}

	// 确定音频格式
	data.audioExt = sniff.AudioExtensionWithFallback(headerBuf, ".mp3")

	// 修复问题：完整读取音频数据到内存，避免流生命周期问题
	header := bytes.NewReader(headerBuf)
	audioStream := io.MultiReader(header, dec)

	// 将完整的音频数据读取到内存中
	data.audioData, err = io.ReadAll(audioStream)
	if err != nil {
		return fmt.Errorf("读取音频数据失败: %w", err)
	}

	// 处理元数据
	if bp.baseProcessor.updateMetadata {
		// 创建元数据处理上下文
		ctx, cancel := context.WithTimeout(context.Background(), metadataTimeout)
		defer cancel()

		var rawMeta common.AudioMeta

		if audioMetaGetter, ok := dec.(common.AudioMetaGetter); ok {
			// 修复问题：为元数据处理创建临时文件，使用audioData而不是audio流
			audioReader := bytes.NewReader(data.audioData)
			tempFile, err := utils.OptimizedWriteTempFile(audioReader, data.audioExt)
			if err != nil {
				bp.logger.Warn("为元数据处理创建临时文件失败",
					zap.String("文件", data.task.InputPath),
					zap.Error(err))
			} else {
				// 获取音频元数据
				meta, err := audioMetaGetter.GetAudioMeta(ctx)
				if err != nil {
					bp.logger.Warn("获取音频元数据失败，将跳过元数据处理",
						zap.String("文件", data.task.InputPath),
						zap.Error(err))
					os.Remove(tempFile) // 立即清理临时文件
				} else {
					rawMeta = meta
					// 存储临时文件路径，用于后续的ffmpeg处理
					data.tempAudioFile = tempFile
					bp.logger.Debug("获取音频元数据成功",
						zap.String("文件", data.task.InputPath))
				}
			}
		}

		// 用文件名信息包装元数据，确保标题准确性（与单文件模式保持一致）
		data.metadata = bp.baseProcessor.processMetadataWithFilename(rawMeta, data.task.InputPath, nil, bp.logger)

		// 存储封面获取器，延迟到写入阶段再获取封面（节省内存）
		if coverGetter, ok := dec.(common.CoverImageGetter); ok {
			data.coverGetter = coverGetter
			bp.logger.Debug("检测到封面图片支持，将在写入阶段获取",
				zap.String("文件", data.task.InputPath))
		}
	}

	// 生成输出路径
	// 修复问题：避免并发安全问题，不修改共享的baseProcessor状态
	// 使用局部变量计算路径，保持与单文件处理的逻辑一致性

	// 确定输出目录和输入目录
	outputDir := data.task.OutputPath
	inputDir := filepath.Dir(data.task.InputPath)

	if outputDir == "" {
		// 源目录模式：使用源文件目录
		outputDir = inputDir
	}

	// 计算相对路径（保持与单文件处理一致）
	inputRelDir, err := filepath.Rel(inputDir, filepath.Dir(data.task.InputPath))
	if err != nil {
		inputRelDir = ""
	}

	inFilename := strings.TrimSuffix(filepath.Base(data.task.InputPath), decoderFactory.Suffix)
	outFilename := bp.baseProcessor.generateOutputFilename(inFilename, data.audioExt)
	data.outputPath = filepath.Join(outputDir, inputRelDir, outFilename)

	bp.logger.Debug("解密完成",
		zap.String("输入文件", data.task.InputPath),
		zap.String("输出路径", data.outputPath),
		zap.String("音频格式", data.audioExt))

	return nil
}

// performWrite 执行写入操作
func (bp *batchProcessor) performWrite(data *pipelineData) error {
	// 清理临时文件
	if data.tempAudioFile != "" {
		defer func() {
			if err := os.Remove(data.tempAudioFile); err != nil {
				bp.logger.Warn("清理临时音频文件失败",
					zap.String("文件", data.tempAudioFile),
					zap.Error(err))
			}
		}()
	}

	// 确保输出目录存在
	outputDir := filepath.Dir(data.outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 检查是否需要覆盖输出文件
	if !bp.baseProcessor.overwriteOutput {
		if _, err := os.Stat(data.outputPath); err == nil {
			bp.logger.Warn("输出文件已存在，跳过", zap.String("文件", data.outputPath))
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("检查输出文件状态失败: %w", err)
		}
	}

	// 如果没有元数据，直接写入音频文件
	if data.metadata == nil {
		outFile, err := os.OpenFile(data.outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("创建输出文件失败: %w", err)
		}
		defer outFile.Close()

		// 修复问题：使用audioData而不是audio流，确保数据完整性
		if _, err := outFile.Write(data.audioData); err != nil {
			return fmt.Errorf("写入音频数据失败: %w", err)
		}
	} else {
		// 使用ffmpeg更新元数据
		ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
		defer cancel()

		var audioFilePath string
		var shouldCleanup bool

		// 如果已经有临时文件，直接使用；否则创建新的
		if data.tempAudioFile != "" {
			audioFilePath = data.tempAudioFile
			shouldCleanup = false // 已经在defer中处理清理
		} else {
			// 修复问题：使用audioData创建临时文件，而不是audio流
			audioReader := bytes.NewReader(data.audioData)
			tempFile, err := utils.OptimizedWriteTempFile(audioReader, data.audioExt)
			if err != nil {
				return fmt.Errorf("写入临时音频文件失败: %w", err)
			}
			audioFilePath = tempFile
			shouldCleanup = true
			defer func() {
				if shouldCleanup {
					os.Remove(tempFile)
				}
			}()
		}

		// 延迟获取封面图片（仅在需要时获取，节省内存）
		if data.coverGetter != nil && data.albumArt == nil {
			if cover, err := data.coverGetter.GetCoverImage(ctx); err != nil {
				bp.logger.Debug("延迟获取封面图片失败",
					zap.String("文件", data.task.InputPath),
					zap.Error(err))
			} else if imgExt, ok := sniff.ImageExtension(cover); !ok {
				bp.logger.Debug("延迟识别封面图片格式失败",
					zap.String("文件", data.task.InputPath))
			} else {
				data.albumArt = cover
				data.albumArtExt = imgExt
				bp.logger.Debug("延迟获取封面图片成功",
					zap.String("文件", data.task.InputPath),
					zap.String("格式", imgExt))
			}
		}

		params := &ffmpeg.UpdateMetadataParams{
			Audio:       audioFilePath,
			AudioExt:    data.audioExt,
			Meta:        data.metadata,
			AlbumArt:    data.albumArt,
			AlbumArtExt: data.albumArtExt,
		}

		if err := ffmpeg.UpdateMeta(ctx, data.outputPath, params, bp.logger); err != nil {
			return fmt.Errorf("更新元数据失败: %w", err)
		}
	}

	bp.logger.Info("文件写入成功",
		zap.String("输入", data.task.InputPath),
		zap.String("输出", data.outputPath))

	return nil
}
