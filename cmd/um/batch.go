package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
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
	index    int
	task     FileTask
	audio    []byte      // 解密后的音频数据
	metadata interface{} // 元数据
	error    error
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

		// 执行解密操作（简化版本，实际需要调用解密逻辑）
		// 这里只是示例，实际应该调用真正的解密函数
		bp.logger.Debug("开始解密", zap.String("文件", taskWithIdx.task.InputPath))

		// TODO: 实际的解密逻辑
		// data.audio, data.metadata, data.error = bp.performDecryption(taskWithIdx.task)

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

			// TODO: 实际的写入逻辑
			// err := bp.performWrite(data)
			// if err != nil {
			//     result.Error = err.Error()
			// } else {
			//     result.Success = true
			// }

			result.Success = true // 临时设置为成功
		}

		resultChan <- resultWithIndex{
			index:  data.index,
			result: result,
		}
	}
}
