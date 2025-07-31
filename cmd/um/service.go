package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Microsoft/go-winio"
	"go.uber.org/zap"
)

// ServiceMessage 服务消息结构
type ServiceMessage struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// ServiceResponse 服务响应结构
type ServiceResponse struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Success   bool        `json:"success"`
	Data      interface{} `json:"data"`
	Error     string      `json:"error,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// Session 会话结构
type Session struct {
	ID         string          `json:"id"`
	CreatedAt  time.Time       `json:"created_at"`
	LastActive time.Time       `json:"last_active"`
	Processor  *batchProcessor `json:"-"`
	Files      []FileTask      `json:"files"`
	Status     string          `json:"status"`
	Options    ProcessOptions  `json:"options"`
	mutex      sync.RWMutex    `json:"-"`
}

// UMService 音乐解密服务
type UMService struct {
	logger   *zap.Logger
	sessions map[string]*Session
	mutex    sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	listener net.Listener
}

// NewUMService 创建新的服务实例
func NewUMService(logger *zap.Logger) *UMService {
	ctx, cancel := context.WithCancel(context.Background())
	return &UMService{
		logger:   logger,
		sessions: make(map[string]*Session),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start 启动服务
func (s *UMService) Start(pipeName string) error {
	// 确定监听地址
	addr := s.getListenAddress(pipeName)

	var err error
	if runtime.GOOS == "windows" {
		// Windows命名管道
		s.listener, err = winio.ListenPipe(addr, nil)
	} else {
		// Unix域套接字
		// 清理可能存在的旧套接字文件
		if _, err := os.Stat(addr); err == nil {
			os.Remove(addr)
		}
		s.listener, err = net.Listen("unix", addr)
	}

	if err != nil {
		return fmt.Errorf("启动服务监听失败: %w", err)
	}

	s.logger.Info("服务启动成功", zap.String("地址", addr))

	// 启动清理协程
	go s.cleanupSessions()

	// 接受连接
	for {
		select {
		case <-s.ctx.Done():
			return nil
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				s.logger.Error("接受连接失败", zap.Error(err))
				continue
			}

			go s.handleConnection(conn)
		}
	}
}

// Stop 停止服务
func (s *UMService) Stop() error {
	s.cancel()
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// getListenAddress 获取监听地址
func (s *UMService) getListenAddress(pipeName string) string {
	if pipeName != "" {
		return pipeName
	}

	if runtime.GOOS == "windows" {
		return `\\.\pipe\um_service`
	}
	return "/tmp/um_service.sock"
}

// handleConnection 处理客户端连接
func (s *UMService) handleConnection(conn net.Conn) {
	defer conn.Close()

	s.logger.Info("新客户端连接", zap.String("远程地址", conn.RemoteAddr().String()))

	scanner := bufio.NewScanner(conn)
	encoder := json.NewEncoder(conn)

	for scanner.Scan() {
		var msg ServiceMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			s.sendError(encoder, "", "解析消息失败", err)
			continue
		}

		response := s.handleMessage(&msg)
		if err := encoder.Encode(response); err != nil {
			s.logger.Error("发送响应失败", zap.Error(err))
			break
		}
	}

	if err := scanner.Err(); err != nil {
		s.logger.Error("读取连接数据失败", zap.Error(err))
	}
}

// handleMessage 处理消息
func (s *UMService) handleMessage(msg *ServiceMessage) *ServiceResponse {
	switch msg.Type {
	case "start_session":
		return s.handleStartSession(msg)
	case "add_files":
		return s.handleAddFiles(msg)
	case "start_processing":
		return s.handleStartProcessing(msg)
	case "get_progress":
		return s.handleGetProgress(msg)
	case "stop_processing":
		return s.handleStopProcessing(msg)
	case "end_session":
		return s.handleEndSession(msg)
	default:
		return s.createErrorResponse(msg.ID, "未知消息类型", nil)
	}
}

// handleStartSession 处理启动会话
func (s *UMService) handleStartSession(msg *ServiceMessage) *ServiceResponse {
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())

	session := &Session{
		ID:         sessionID,
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
		Files:      make([]FileTask, 0),
		Status:     "created",
		Options:    ProcessOptions{}, // 使用默认选项
	}

	s.mutex.Lock()
	s.sessions[sessionID] = session
	s.mutex.Unlock()

	s.logger.Info("创建新会话", zap.String("会话ID", sessionID))

	return &ServiceResponse{
		ID:        msg.ID,
		Type:      "session_started",
		Success:   true,
		Data:      map[string]string{"session_id": sessionID},
		Timestamp: time.Now().Unix(),
	}
}

// handleAddFiles 处理添加文件
func (s *UMService) handleAddFiles(msg *ServiceMessage) *ServiceResponse {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return s.createErrorResponse(msg.ID, "无效的消息数据格式", nil)
	}

	sessionID, ok := data["session_id"].(string)
	if !ok {
		return s.createErrorResponse(msg.ID, "缺少会话ID", nil)
	}

	filesData, ok := data["files"].([]interface{})
	if !ok {
		return s.createErrorResponse(msg.ID, "无效的文件列表格式", nil)
	}

	// 获取会话
	s.mutex.RLock()
	session, exists := s.sessions[sessionID]
	s.mutex.RUnlock()

	if !exists {
		return s.createErrorResponse(msg.ID, "会话不存在", nil)
	}

	// 解析文件列表
	var files []FileTask
	for _, fileData := range filesData {
		fileMap, ok := fileData.(map[string]interface{})
		if !ok {
			continue
		}

		inputPath, ok := fileMap["input_path"].(string)
		if !ok {
			continue
		}

		task := FileTask{
			InputPath: inputPath,
		}

		if outputPath, ok := fileMap["output_path"].(string); ok {
			task.OutputPath = outputPath
		}

		files = append(files, task)
	}

	// 验证文件格式和存在性
	validFiles := make([]FileTask, 0, len(files))
	for _, file := range files {
		if _, err := os.Stat(file.InputPath); err != nil {
			s.logger.Warn("文件不存在", zap.String("路径", file.InputPath))
			continue
		}

		// 检查文件格式（简单检查扩展名）
		ext := strings.ToLower(filepath.Ext(file.InputPath))
		if s.isSupportedFormat(ext) {
			validFiles = append(validFiles, file)
		} else {
			s.logger.Warn("不支持的文件格式", zap.String("路径", file.InputPath), zap.String("扩展名", ext))
		}
	}

	// 更新会话
	session.mutex.Lock()
	session.Files = append(session.Files, validFiles...)
	session.LastActive = time.Now()
	session.mutex.Unlock()

	s.logger.Info("添加文件到会话",
		zap.String("会话ID", sessionID),
		zap.Int("有效文件数", len(validFiles)),
		zap.Int("总文件数", len(files)))

	return s.createSuccessResponse(msg.ID, "files_added", map[string]interface{}{
		"added_count": len(validFiles),
		"total_files": len(session.Files),
	})
}

// handleStartProcessing 处理开始处理
func (s *UMService) handleStartProcessing(msg *ServiceMessage) *ServiceResponse {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return s.createErrorResponse(msg.ID, "无效的消息数据格式", nil)
	}

	sessionID, ok := data["session_id"].(string)
	if !ok {
		return s.createErrorResponse(msg.ID, "缺少会话ID", nil)
	}

	// 获取会话
	s.mutex.RLock()
	session, exists := s.sessions[sessionID]
	s.mutex.RUnlock()

	if !exists {
		return s.createErrorResponse(msg.ID, "会话不存在", nil)
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	// 检查会话状态
	if session.Status == "processing" {
		return s.createErrorResponse(msg.ID, "会话正在处理中", nil)
	}

	if len(session.Files) == 0 {
		return s.createErrorResponse(msg.ID, "没有文件需要处理", nil)
	}

	// 解析处理选项
	options := ProcessOptions{
		RemoveSource:    false,
		UpdateMetadata:  true,
		OverwriteOutput: true,
		SkipNoop:        true,
		NamingFormat:    "auto",
	}

	if optionsData, ok := data["options"].(map[string]interface{}); ok {
		if removeSource, ok := optionsData["remove_source"].(bool); ok {
			options.RemoveSource = removeSource
		}
		if updateMetadata, ok := optionsData["update_metadata"].(bool); ok {
			options.UpdateMetadata = updateMetadata
		}
		if overwriteOutput, ok := optionsData["overwrite_output"].(bool); ok {
			options.OverwriteOutput = overwriteOutput
		}
		if skipNoop, ok := optionsData["skip_noop"].(bool); ok {
			options.SkipNoop = skipNoop
		}
		if namingFormat, ok := optionsData["naming_format"].(string); ok {
			options.NamingFormat = namingFormat
		}
	}

	// 更新会话状态
	session.Status = "processing"
	session.Options = options
	session.LastActive = time.Now()

	// 创建批处理器
	session.Processor = newBatchProcessor(options, s.logger)

	// 启动异步处理
	go s.processSessionFiles(sessionID)

	s.logger.Info("开始处理会话文件",
		zap.String("会话ID", sessionID),
		zap.Int("文件数量", len(session.Files)))

	return s.createSuccessResponse(msg.ID, "processing_started", map[string]interface{}{
		"session_id": sessionID,
		"file_count": len(session.Files),
		"status":     "processing",
	})
}

// handleGetProgress 处理获取进度
func (s *UMService) handleGetProgress(msg *ServiceMessage) *ServiceResponse {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return s.createErrorResponse(msg.ID, "无效的消息数据格式", nil)
	}

	sessionID, ok := data["session_id"].(string)
	if !ok {
		return s.createErrorResponse(msg.ID, "缺少会话ID", nil)
	}

	// 获取会话
	s.mutex.RLock()
	session, exists := s.sessions[sessionID]
	s.mutex.RUnlock()

	if !exists {
		return s.createErrorResponse(msg.ID, "会话不存在", nil)
	}

	session.mutex.RLock()
	status := session.Status
	totalFiles := len(session.Files)
	session.mutex.RUnlock()

	// 计算进度
	progress := 0.0
	processedFiles := 0
	currentFile := ""

	if session.Processor != nil {
		// 这里需要从批处理器获取实际进度
		// 暂时使用简单的状态判断
		switch status {
		case "processing":
			progress = 50.0 // 处理中
		case "completed":
			progress = 100.0
			processedFiles = totalFiles
		case "error":
			progress = 0.0
		default:
			progress = 0.0
		}
	}

	return s.createSuccessResponse(msg.ID, "progress_update", map[string]interface{}{
		"session_id":      sessionID,
		"progress":        progress,
		"status":          status,
		"total_files":     totalFiles,
		"processed_files": processedFiles,
		"current_file":    currentFile,
	})
}

// handleStopProcessing 处理停止处理
func (s *UMService) handleStopProcessing(msg *ServiceMessage) *ServiceResponse {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return s.createErrorResponse(msg.ID, "无效的消息数据格式", nil)
	}

	sessionID, ok := data["session_id"].(string)
	if !ok {
		return s.createErrorResponse(msg.ID, "缺少会话ID", nil)
	}

	// 获取会话
	s.mutex.RLock()
	session, exists := s.sessions[sessionID]
	s.mutex.RUnlock()

	if !exists {
		return s.createErrorResponse(msg.ID, "会话不存在", nil)
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	// 检查会话状态
	if session.Status != "processing" {
		return s.createErrorResponse(msg.ID, "会话未在处理中", nil)
	}

	// 停止处理
	session.Status = "stopped"
	session.LastActive = time.Now()

	// 这里应该停止批处理器，但当前批处理器没有停止机制
	// 可以通过context来实现，暂时只更新状态

	s.logger.Info("停止处理会话", zap.String("会话ID", sessionID))

	return s.createSuccessResponse(msg.ID, "processing_stopped", map[string]interface{}{
		"session_id": sessionID,
		"status":     "stopped",
	})
}

// handleEndSession 处理结束会话
func (s *UMService) handleEndSession(msg *ServiceMessage) *ServiceResponse {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return s.createErrorResponse(msg.ID, "无效的消息数据格式", nil)
	}

	sessionID, ok := data["session_id"].(string)
	if !ok {
		return s.createErrorResponse(msg.ID, "缺少会话ID", nil)
	}

	// 获取并删除会话
	s.mutex.Lock()
	session, exists := s.sessions[sessionID]
	if exists {
		delete(s.sessions, sessionID)
	}
	s.mutex.Unlock()

	if !exists {
		return s.createErrorResponse(msg.ID, "会话不存在", nil)
	}

	// 清理会话资源
	session.mutex.Lock()
	session.Status = "ended"
	session.Files = nil
	session.Processor = nil
	session.mutex.Unlock()

	s.logger.Info("结束会话", zap.String("会话ID", sessionID))

	return s.createSuccessResponse(msg.ID, "session_ended", map[string]interface{}{
		"session_id": sessionID,
		"status":     "ended",
	})
}

// cleanupSessions 清理过期会话
func (s *UMService) cleanupSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.performCleanup()
		}
	}
}

// performCleanup 执行清理
func (s *UMService) performCleanup() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	for id, session := range s.sessions {
		if now.Sub(session.LastActive) > 30*time.Minute {
			delete(s.sessions, id)
			s.logger.Info("清理过期会话", zap.String("会话ID", id))
		}
	}
}

// 辅助方法
func (s *UMService) createSuccessResponse(id, responseType string, data interface{}) *ServiceResponse {
	return &ServiceResponse{
		ID:        id,
		Type:      responseType,
		Success:   true,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
}

func (s *UMService) createErrorResponse(id, errorMsg string, err error) *ServiceResponse {
	errorText := errorMsg
	if err != nil {
		errorText = fmt.Sprintf("%s: %v", errorMsg, err)
	}

	return &ServiceResponse{
		ID:        id,
		Type:      "error",
		Success:   false,
		Error:     errorText,
		Timestamp: time.Now().Unix(),
	}
}

func (s *UMService) sendError(encoder *json.Encoder, id, msg string, err error) {
	response := s.createErrorResponse(id, msg, err)
	encoder.Encode(response)
}

// processSessionFiles 异步处理会话文件
func (s *UMService) processSessionFiles(sessionID string) {
	// 获取会话
	s.mutex.RLock()
	session, exists := s.sessions[sessionID]
	s.mutex.RUnlock()

	if !exists {
		s.logger.Error("处理文件时会话不存在", zap.String("会话ID", sessionID))
		return
	}

	session.mutex.Lock()
	files := make([]FileTask, len(session.Files))
	copy(files, session.Files)
	processor := session.Processor
	options := session.Options
	session.mutex.Unlock()

	if processor == nil {
		s.logger.Error("批处理器未初始化", zap.String("会话ID", sessionID))
		session.mutex.Lock()
		session.Status = "error"
		session.mutex.Unlock()
		return
	}

	// 构建批处理请求
	request := &BatchRequest{
		Files:   files,
		Options: options,
	}

	s.logger.Info("开始异步处理文件",
		zap.String("会话ID", sessionID),
		zap.Int("文件数量", len(files)))

	// 执行批处理
	response := processor.processBatch(request)

	// 更新会话状态
	session.mutex.Lock()
	if response.SuccessCount == len(files) {
		session.Status = "completed"
	} else if response.SuccessCount > 0 {
		session.Status = "partial_success"
	} else {
		session.Status = "error"
	}
	session.LastActive = time.Now()
	session.mutex.Unlock()

	s.logger.Info("文件处理完成",
		zap.String("会话ID", sessionID),
		zap.String("状态", session.Status),
		zap.Int("成功", response.SuccessCount),
		zap.Int("失败", response.FailedCount))
}

// isSupportedFormat 检查文件格式是否支持
func (s *UMService) isSupportedFormat(ext string) bool {
	supportedFormats := map[string]bool{
		".ncm":  true, // 网易云音乐
		".qmc":  true, // QQ音乐
		".qmc0": true,
		".qmc3": true,
		".kgm":  true, // 酷狗音乐
		".kgma": true,
		".kwm":  true, // 酷我音乐
		".tm0":  true, // 天天动听
		".tm2":  true,
		".tm3":  true,
		".tm6":  true,
		".xm":   true, // 虾米音乐
		".x2m":  true,
		".x3m":  true,
	}
	return supportedFormats[ext]
}

// runServiceMode 运行服务模式
func runServiceMode(logger *zap.Logger, pipeName string) error {
	logger.Info("启动服务模式", zap.String("管道名称", pipeName))

	service := NewUMService(logger)

	// 启动服务
	return service.Start(pipeName)
}
