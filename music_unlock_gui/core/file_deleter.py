#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
文件删除核心逻辑模块
"""

import os
import threading
import queue
from typing import List, Tuple, Callable, Optional
import logging


class FileDeleter:
    """文件删除器类"""
    
    def __init__(self):
        """初始化文件删除器"""
        self.logger = self._setup_logger()
        self.is_scanning = False
        self.is_deleting = False
        self.cancel_requested = False
    
    def _setup_logger(self) -> logging.Logger:
        """设置日志记录器"""
        logger = logging.getLogger(f"{__name__}.FileDeleter")
        if not logger.handlers:
            handler = logging.StreamHandler()
            formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')
            handler.setFormatter(formatter)
            logger.addHandler(handler)
            logger.setLevel(logging.INFO)
        return logger
    
    def scan_files(self, folder_path: str, selected_extensions: List[str], 
                   progress_callback: Optional[Callable] = None) -> Tuple[bool, List[str], str]:
        """
        扫描指定文件夹中的文件
        
        Args:
            folder_path: 要扫描的文件夹路径
            selected_extensions: 选中的文件扩展名列表
            progress_callback: 进度回调函数
            
        Returns:
            Tuple[bool, List[str], str]: (是否成功, 文件列表, 错误信息)
        """
        if not os.path.exists(folder_path):
            return False, [], f"文件夹不存在: {folder_path}"
        
        if not os.path.isdir(folder_path):
            return False, [], f"路径不是文件夹: {folder_path}"
        
        if not selected_extensions:
            return False, [], "未选择任何文件格式"
        
        self.is_scanning = True
        self.cancel_requested = False
        found_files = []
        
        try:
            # 将扩展名转换为小写集合以便快速查找
            ext_set = {ext.lower() for ext in selected_extensions}
            
            total_dirs = sum(1 for _, dirs, _ in os.walk(folder_path))
            processed_dirs = 0
            
            self.logger.info(f"开始扫描文件夹: {folder_path}")
            self.logger.info(f"目标格式: {selected_extensions}")
            
            for root, dirs, files in os.walk(folder_path):
                if self.cancel_requested:
                    self.logger.info("扫描被取消")
                    break
                
                # 更新进度
                processed_dirs += 1
                if progress_callback:
                    progress = int((processed_dirs / total_dirs) * 100) if total_dirs > 0 else 100
                    progress_callback(f"扫描中... ({processed_dirs}/{total_dirs})", progress)
                
                for file in files:
                    if self.cancel_requested:
                        break
                    
                    # 检查文件扩展名
                    _, file_ext = os.path.splitext(file)
                    if file_ext.lower() in ext_set:
                        file_path = os.path.join(root, file)
                        found_files.append(file_path)
                        self.logger.debug(f"找到文件: {file_path}")
            
            self.logger.info(f"扫描完成，找到 {len(found_files)} 个文件")
            return True, found_files, ""
            
        except Exception as e:
            error_msg = f"扫描文件时出错: {str(e)}"
            self.logger.error(error_msg)
            return False, [], error_msg
        finally:
            self.is_scanning = False
    
    def delete_files(self, file_list: List[str], 
                     progress_callback: Optional[Callable] = None) -> Tuple[int, int, List[str]]:
        """
        删除文件列表中的文件
        
        Args:
            file_list: 要删除的文件路径列表
            progress_callback: 进度回调函数
            
        Returns:
            Tuple[int, int, List[str]]: (成功删除数, 失败删除数, 失败文件列表)
        """
        if not file_list:
            return 0, 0, []
        
        self.is_deleting = True
        self.cancel_requested = False
        
        deleted_count = 0
        failed_count = 0
        failed_files = []
        total_files = len(file_list)
        
        try:
            self.logger.info(f"开始删除 {total_files} 个文件")
            
            for i, file_path in enumerate(file_list):
                if self.cancel_requested:
                    self.logger.info("删除操作被取消")
                    break
                
                # 更新进度
                if progress_callback:
                    progress = int(((i + 1) / total_files) * 100)
                    progress_callback(f"删除中... ({i + 1}/{total_files})", progress)
                
                try:
                    if os.path.exists(file_path):
                        os.remove(file_path)
                        deleted_count += 1
                        self.logger.debug(f"已删除: {file_path}")
                    else:
                        self.logger.warning(f"文件不存在，跳过: {file_path}")
                        
                except OSError as e:
                    failed_count += 1
                    failed_files.append(file_path)
                    self.logger.error(f"删除失败: {file_path} - {e}")
                except Exception as e:
                    failed_count += 1
                    failed_files.append(file_path)
                    self.logger.error(f"删除时发生未知错误: {file_path} - {e}")
            
            self.logger.info(f"删除完成: 成功 {deleted_count}, 失败 {failed_count}")
            return deleted_count, failed_count, failed_files
            
        except Exception as e:
            self.logger.error(f"删除操作出错: {str(e)}")
            return deleted_count, failed_count, failed_files
        finally:
            self.is_deleting = False
    
    def cancel_operation(self):
        """取消当前操作"""
        self.cancel_requested = True
        self.logger.info("请求取消操作")
    
    def is_busy(self) -> bool:
        """检查是否正在执行操作"""
        return self.is_scanning or self.is_deleting


class ThreadedFileDeleter:
    """线程化文件删除器"""
    
    def __init__(self):
        """初始化线程化文件删除器"""
        self.deleter = FileDeleter()
        self.message_queue = queue.Queue()
        self.current_thread = None
    
    def scan_files_async(self, folder_path: str, selected_extensions: List[str]):
        """异步扫描文件"""
        if self.current_thread and self.current_thread.is_alive():
            return False
        
        def scan_worker():
            def progress_callback(message, progress):
                self.message_queue.put(('progress', message, progress))
            
            success, files, error = self.deleter.scan_files(folder_path, selected_extensions, progress_callback)
            self.message_queue.put(('scan_complete', success, files, error))
        
        self.current_thread = threading.Thread(target=scan_worker, daemon=True)
        self.current_thread.start()
        return True
    
    def delete_files_async(self, file_list: List[str]):
        """异步删除文件"""
        if self.current_thread and self.current_thread.is_alive():
            return False
        
        def delete_worker():
            def progress_callback(message, progress):
                self.message_queue.put(('progress', message, progress))
            
            deleted, failed, failed_files = self.deleter.delete_files(file_list, progress_callback)
            self.message_queue.put(('delete_complete', deleted, failed, failed_files))
        
        self.current_thread = threading.Thread(target=delete_worker, daemon=True)
        self.current_thread.start()
        return True
    
    def cancel_operation(self):
        """取消当前操作"""
        self.deleter.cancel_operation()
    
    def is_busy(self) -> bool:
        """检查是否正在执行操作"""
        return self.deleter.is_busy()
    
    def get_message_queue(self) -> queue.Queue:
        """获取消息队列"""
        return self.message_queue
