#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
线程管理器 - 负责管理多线程文件处理
"""

import threading
import queue
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import List, Callable, Optional
import logging


class ThreadManager:
    """线程管理器类"""
    
    def __init__(self, max_workers: int = 6):
        """
        初始化线程管理器
        
        Args:
            max_workers: 最大工作线程数
        """
        self.max_workers = max_workers
        self.executor: Optional[ThreadPoolExecutor] = None
        self.futures = []
        self.stop_event = threading.Event()
        self.logger = self._setup_logger()
        self.processing = False
    
    def _setup_logger(self) -> logging.Logger:
        """设置日志记录器"""
        logger = logging.getLogger('ThreadManager')
        logger.setLevel(logging.INFO)
        
        if not logger.handlers:
            handler = logging.StreamHandler()
            formatter = logging.Formatter(
                '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
            )
            handler.setFormatter(formatter)
            logger.addHandler(handler)
        
        return logger
    
    def start_processing(self, file_list: List[str], output_dir: str = None,
                        processor=None, message_queue: queue.Queue = None,
                        use_source_dir: bool = False):
        """
        开始处理文件列表

        Args:
            file_list: 要处理的文件列表
            output_dir: 输出目录（可选）
            processor: 文件处理器实例
            message_queue: 消息队列，用于向GUI线程发送状态更新
            use_source_dir: 是否使用源文件目录作为输出目录
        """
        if self.processing:
            self.logger.warning("已有处理任务在运行")
            return
        
        self.processing = True
        self.stop_event.clear()
        
        # 创建线程池
        self.executor = ThreadPoolExecutor(max_workers=self.max_workers)
        self.futures = []
        
        self.logger.info(f"开始处理 {len(file_list)} 个文件，使用 {self.max_workers} 个工作线程")
        
        # 提交所有任务
        for file_path in file_list:
            if self.stop_event.is_set():
                break
                
            future = self.executor.submit(
                self._process_single_file,
                file_path,
                output_dir,
                processor,
                message_queue,
                use_source_dir
            )
            self.futures.append(future)
        
        # 启动监控线程
        monitor_thread = threading.Thread(
            target=self._monitor_completion,
            args=(message_queue,),
            daemon=True
        )
        monitor_thread.start()
    
    def _process_single_file(self, file_path: str, output_dir: str = None,
                           processor=None, message_queue: queue.Queue = None,
                           use_source_dir: bool = False):
        """
        处理单个文件的工作函数

        Args:
            file_path: 文件路径
            output_dir: 输出目录（可选）
            processor: 文件处理器实例
            message_queue: 消息队列
            use_source_dir: 是否使用源文件目录
        """
        try:
            if self.stop_event.is_set():
                return
            
            self.logger.info(f"开始处理文件: {file_path}")
            
            # 进度回调函数
            def progress_callback(progress: int):
                if not self.stop_event.is_set():
                    message_queue.put({
                        'type': 'progress',
                        'file_path': file_path,
                        'progress': progress
                    })
            
            # 处理文件
            success, message = processor.process_file(
                file_path,
                output_dir,
                progress_callback,
                use_source_dir
            )
            
            if self.stop_event.is_set():
                return
            
            # 发送结果消息
            if success:
                message_queue.put({
                    'type': 'success',
                    'file_path': file_path,
                    'message': message
                })
                self.logger.info(f"文件处理成功: {file_path}")
            else:
                message_queue.put({
                    'type': 'error',
                    'file_path': file_path,
                    'message': message
                })
                self.logger.error(f"文件处理失败: {file_path}, 错误: {message}")
                
        except Exception as e:
            if not self.stop_event.is_set():
                error_msg = f"处理文件时发生异常: {str(e)}"
                message_queue.put({
                    'type': 'error',
                    'file_path': file_path,
                    'message': error_msg
                })
                self.logger.error(f"文件处理异常: {file_path}, 错误: {error_msg}")
    
    def _monitor_completion(self, message_queue: queue.Queue):
        """
        监控所有任务完成情况
        
        Args:
            message_queue: 消息队列
        """
        try:
            # 等待所有任务完成
            completed_count = 0
            total_count = len(self.futures)
            
            for future in as_completed(self.futures):
                if self.stop_event.is_set():
                    break
                
                try:
                    future.result()  # 获取结果，如果有异常会抛出
                except Exception as e:
                    self.logger.error(f"任务执行异常: {str(e)}")
                
                completed_count += 1
                self.logger.debug(f"任务完成进度: {completed_count}/{total_count}")
            
            # 所有任务完成
            if not self.stop_event.is_set():
                message_queue.put({
                    'type': 'all_complete',
                    'completed': completed_count,
                    'total': total_count
                })
                self.logger.info(f"所有任务完成: {completed_count}/{total_count}")
            
        except Exception as e:
            self.logger.error(f"监控任务完成时发生异常: {str(e)}")
        finally:
            self.processing = False
            if self.executor:
                self.executor.shutdown(wait=False)
    
    def stop_all(self):
        """停止所有正在进行的任务"""
        if not self.processing:
            return
        
        self.logger.info("正在停止所有任务...")
        self.stop_event.set()
        
        # 取消所有未开始的任务
        for future in self.futures:
            future.cancel()
        
        # 关闭线程池
        if self.executor:
            self.executor.shutdown(wait=False)
        
        self.processing = False
        self.logger.info("所有任务已停止")
    
    def is_processing(self) -> bool:
        """
        检查是否正在处理任务
        
        Returns:
            bool: 是否正在处理
        """
        return self.processing
    
    def get_active_count(self) -> int:
        """
        获取活跃线程数
        
        Returns:
            int: 活跃线程数
        """
        if self.executor:
            return self.executor._threads.__len__()
        return 0
    
    def get_pending_count(self) -> int:
        """
        获取待处理任务数
        
        Returns:
            int: 待处理任务数
        """
        if not self.futures:
            return 0
        
        pending_count = 0
        for future in self.futures:
            if not future.done():
                pending_count += 1
        
        return pending_count
    
    def get_completed_count(self) -> int:
        """
        获取已完成任务数
        
        Returns:
            int: 已完成任务数
        """
        if not self.futures:
            return 0
        
        completed_count = 0
        for future in self.futures:
            if future.done():
                completed_count += 1
        
        return completed_count
    
    def get_status_summary(self) -> dict:
        """
        获取状态摘要
        
        Returns:
            dict: 状态信息
        """
        return {
            'processing': self.processing,
            'total_tasks': len(self.futures) if self.futures else 0,
            'completed_tasks': self.get_completed_count(),
            'pending_tasks': self.get_pending_count(),
            'active_threads': self.get_active_count(),
            'max_workers': self.max_workers
        }
