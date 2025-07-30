package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
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

// batchProcessor 批处理器
type batchProcessor struct {
	logger  *zap.Logger
	options ProcessOptions

	// 复用的processor实例，避免重复初始化
	baseProcessor *processor

	// 并发控制
	maxWorkers int
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

	// 设置并发数，默认为CPU核心数
	maxWorkers := runtime.NumCPU()
	if maxWorkers > 8 {
		maxWorkers = 8 // 限制最大并发数
	}

	return &batchProcessor{
		logger:        logger,
		options:       options,
		baseProcessor: baseProc,
		maxWorkers:    maxWorkers,
	}
}

// processBatch 处理批量任务（并发版本）
func (bp *batchProcessor) processBatch(request *BatchRequest) *BatchResponse {
	startTime := time.Now()

	response := &BatchResponse{
		Results:    make([]ProcessResult, len(request.Files)),
		TotalFiles: len(request.Files),
	}

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
