#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
音乐解密服务客户端
实现与Go服务端的IPC通信
"""

import json
import logging
import os
import platform
import socket
import threading
import time
import uuid
from typing import Dict, List, Optional, Callable, Any


class ServiceClient:
    """音乐解密服务客户端"""
    
    def __init__(self, service_path: str = None):
        """
        初始化服务客户端
        
        Args:
            service_path: 服务路径（Windows命名管道或Unix套接字）
        """
        self.logger = self._setup_logger()
        self.is_windows = platform.system() == "Windows"
        self.service_path = service_path or self._get_default_service_path()
        self.socket = None
        self.connected = False
        self.session_id = None
        self._lock = threading.Lock()
        
    def _setup_logger(self) -> logging.Logger:
        """设置日志记录器"""
        logger = logging.getLogger(f"{__name__}.ServiceClient")
        if not logger.handlers:
            handler = logging.StreamHandler()
            formatter = logging.Formatter(
                '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
            )
            handler.setFormatter(formatter)
            logger.addHandler(handler)
            logger.setLevel(logging.INFO)
        return logger
        
    def _get_default_service_path(self) -> str:
        """获取默认服务路径"""
        if self.is_windows:
            return r'\\.\pipe\um_service'
        else:
            return '/tmp/um_service.sock'
            
    def connect(self, timeout: float = 10.0) -> bool:
        """
        连接到服务
        
        Args:
            timeout: 连接超时时间（秒）
            
        Returns:
            bool: 是否连接成功
        """
        with self._lock:
            if self.connected:
                return True
                
            try:
                if self.is_windows:
                    # Windows命名管道
                    import win32file
                    import win32pipe
                    
                    # 等待管道可用
                    win32pipe.WaitNamedPipe(self.service_path, int(timeout * 1000))
                    
                    self.socket = win32file.CreateFile(
                        self.service_path,
                        win32file.GENERIC_READ | win32file.GENERIC_WRITE,
                        0,
                        None,
                        win32file.OPEN_EXISTING,
                        0,
                        None
                    )
                else:
                    # Unix域套接字
                    self.socket = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
                    self.socket.settimeout(timeout)
                    self.socket.connect(self.service_path)
                    
                self.connected = True
                self.logger.info(f"成功连接到服务: {self.service_path}")
                return True
                
            except Exception as e:
                self.logger.error(f"连接服务失败: {e}")
                self.socket = None
                self.connected = False
                return False
                
    def disconnect(self):
        """断开连接"""
        with self._lock:
            if self.socket:
                try:
                    if self.is_windows:
                        import win32file
                        win32file.CloseHandle(self.socket)
                    else:
                        self.socket.close()
                except Exception as e:
                    self.logger.error(f"断开连接时出错: {e}")
                finally:
                    self.socket = None
                    self.connected = False
                    self.session_id = None
                    self.logger.info("已断开服务连接")
                    
    def _send_message(self, msg_type: str, data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        发送消息到服务端
        
        Args:
            msg_type: 消息类型
            data: 消息数据
            
        Returns:
            Dict: 服务端响应，失败时返回None
        """
        if not self.connected:
            self.logger.error("未连接到服务")
            return None
            
        message = {
            "id": str(uuid.uuid4()),
            "type": msg_type,
            "data": data,
            "timestamp": int(time.time())
        }
        
        try:
            # 发送JSON消息
            json_str = json.dumps(message, ensure_ascii=False)
            message_bytes = (json_str + '\n').encode('utf-8')

            if self.is_windows:
                import win32file
                win32file.WriteFile(self.socket, message_bytes)
            else:
                self.socket.send(message_bytes)
            
            self.logger.debug(f"发送消息: {msg_type}")

            # 接收响应
            if self.is_windows:
                import win32file
                result, response_data = win32file.ReadFile(self.socket, 4096)
            else:
                response_data = self.socket.recv(4096)

            if response_data:
                response_str = response_data.decode('utf-8').strip()
                response = json.loads(response_str)
                self.logger.debug(f"收到响应: {response.get('type', 'unknown')}")
                return response
            else:
                self.logger.error("未收到响应")
                return None
                
        except Exception as e:
            self.logger.error(f"发送消息失败: {e}")
            return None
            
    def start_session(self) -> bool:
        """
        启动处理会话
        
        Returns:
            bool: 是否成功启动会话
        """
        response = self._send_message("start_session", {})
        
        if response and response.get("success"):
            self.session_id = response.get("data", {}).get("session_id")
            self.logger.info(f"启动会话成功: {self.session_id}")
            return True
        else:
            error = response.get("error", "未知错误") if response else "无响应"
            self.logger.error(f"启动会话失败: {error}")
            return False
            
    def add_files(self, files: List[Dict[str, str]]) -> bool:
        """
        添加文件到处理队列
        
        Args:
            files: 文件列表，每个文件包含input_path和可选的output_path
            
        Returns:
            bool: 是否成功添加文件
        """
        if not self.session_id:
            self.logger.error("未启动会话")
            return False
            
        data = {
            "session_id": self.session_id,
            "files": files
        }
        
        response = self._send_message("add_files", data)
        
        if response and response.get("success"):
            added_count = response.get("data", {}).get("added_count", 0)
            self.logger.info(f"成功添加 {added_count} 个文件")
            return True
        else:
            error = response.get("error", "未知错误") if response else "无响应"
            self.logger.error(f"添加文件失败: {error}")
            return False
            
    def start_processing(self, options: Dict[str, Any] = None) -> bool:
        """
        开始处理文件
        
        Args:
            options: 处理选项
            
        Returns:
            bool: 是否成功开始处理
        """
        if not self.session_id:
            self.logger.error("未启动会话")
            return False
            
        if options is None:
            options = {
                "remove_source": False,
                "update_metadata": False,
                "overwrite_output": True,
                "skip_noop": True,
                "naming_format": "auto"
            }
            
        data = {
            "session_id": self.session_id,
            "options": options
        }
        
        response = self._send_message("start_processing", data)
        
        if response and response.get("success"):
            self.logger.info("开始处理文件")
            return True
        else:
            error = response.get("error", "未知错误") if response else "无响应"
            self.logger.error(f"开始处理失败: {error}")
            return False
            
    def get_progress(self) -> Optional[Dict[str, Any]]:
        """
        获取处理进度
        
        Returns:
            Dict: 进度信息，失败时返回None
        """
        if not self.session_id:
            self.logger.error("未启动会话")
            return None
            
        data = {"session_id": self.session_id}
        response = self._send_message("get_progress", data)
        
        if response and response.get("success"):
            return response.get("data", {})
        else:
            error = response.get("error", "未知错误") if response else "无响应"
            self.logger.error(f"获取进度失败: {error}")
            return None
            
    def stop_processing(self) -> bool:
        """
        停止处理
        
        Returns:
            bool: 是否成功停止处理
        """
        if not self.session_id:
            self.logger.error("未启动会话")
            return False
            
        data = {"session_id": self.session_id}
        response = self._send_message("stop_processing", data)
        
        if response and response.get("success"):
            self.logger.info("停止处理成功")
            return True
        else:
            error = response.get("error", "未知错误") if response else "无响应"
            self.logger.error(f"停止处理失败: {error}")
            return False
            
    def end_session(self) -> bool:
        """
        结束会话
        
        Returns:
            bool: 是否成功结束会话
        """
        if not self.session_id:
            return True  # 没有会话，认为已结束
            
        data = {"session_id": self.session_id}
        response = self._send_message("end_session", data)
        
        if response and response.get("success"):
            self.logger.info("结束会话成功")
            self.session_id = None
            return True
        else:
            error = response.get("error", "未知错误") if response else "无响应"
            self.logger.error(f"结束会话失败: {error}")
            self.session_id = None  # 强制清除会话ID
            return False
            
    def __enter__(self):
        """上下文管理器入口"""
        self.connect()
        return self
        
    def __exit__(self, exc_type, exc_val, exc_tb):
        """上下文管理器出口"""
        if self.session_id:
            self.end_session()
        self.disconnect()
