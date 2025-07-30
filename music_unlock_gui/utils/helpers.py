#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
工具函数模块
"""

import os
import sys
import platform
import subprocess
from typing import List, Tuple, Optional
import tkinter as tk
from tkinter import messagebox


def get_resource_path(relative_path: str) -> str:
    """
    获取资源文件的绝对路径，支持打包后的环境
    
    Args:
        relative_path: 相对路径
        
    Returns:
        str: 绝对路径
    """
    try:
        # PyInstaller创建的临时文件夹路径
        base_path = sys._MEIPASS
    except AttributeError:
        # 开发环境
        base_path = os.path.abspath(".")
    
    return os.path.join(base_path, relative_path)


def validate_file_path(file_path: str) -> bool:
    """
    验证文件路径是否有效
    
    Args:
        file_path: 文件路径
        
    Returns:
        bool: 是否有效
    """
    if not file_path:
        return False
    
    if not os.path.exists(file_path):
        return False
    
    if not os.path.isfile(file_path):
        return False
    
    return True


def validate_directory_path(dir_path: str) -> bool:
    """
    验证目录路径是否有效
    
    Args:
        dir_path: 目录路径
        
    Returns:
        bool: 是否有效
    """
    if not dir_path:
        return False
    
    if not os.path.exists(dir_path):
        return False
    
    if not os.path.isdir(dir_path):
        return False
    
    return True


def format_file_size(size_bytes: int) -> str:
    """
    格式化文件大小显示
    
    Args:
        size_bytes: 字节数
        
    Returns:
        str: 格式化后的大小字符串
    """
    if size_bytes == 0:
        return "0 B"
    
    size_names = ["B", "KB", "MB", "GB", "TB"]
    i = 0
    while size_bytes >= 1024 and i < len(size_names) - 1:
        size_bytes /= 1024.0
        i += 1
    
    return f"{size_bytes:.1f} {size_names[i]}"


def get_file_info(file_path: str) -> dict:
    """
    获取文件信息
    
    Args:
        file_path: 文件路径
        
    Returns:
        dict: 文件信息字典
    """
    try:
        stat = os.stat(file_path)
        return {
            'name': os.path.basename(file_path),
            'path': file_path,
            'size': stat.st_size,
            'size_formatted': format_file_size(stat.st_size),
            'modified': stat.st_mtime,
            'extension': os.path.splitext(file_path)[1].lower()
        }
    except Exception:
        return {
            'name': os.path.basename(file_path),
            'path': file_path,
            'size': 0,
            'size_formatted': '0 B',
            'modified': 0,
            'extension': os.path.splitext(file_path)[1].lower()
        }


def center_window(window: tk.Tk, width: int, height: int):
    """
    将窗口居中显示
    
    Args:
        window: tkinter窗口对象
        width: 窗口宽度
        height: 窗口高度
    """
    # 获取屏幕尺寸
    screen_width = window.winfo_screenwidth()
    screen_height = window.winfo_screenheight()
    
    # 计算居中位置
    x = (screen_width - width) // 2
    y = (screen_height - height) // 2
    
    # 设置窗口位置
    window.geometry(f"{width}x{height}+{x}+{y}")


def show_error_dialog(title: str, message: str, parent=None):
    """
    显示错误对话框
    
    Args:
        title: 对话框标题
        message: 错误消息
        parent: 父窗口
    """
    messagebox.showerror(title, message, parent=parent)


def show_info_dialog(title: str, message: str, parent=None):
    """
    显示信息对话框
    
    Args:
        title: 对话框标题
        message: 信息消息
        parent: 父窗口
    """
    messagebox.showinfo(title, message, parent=parent)


def show_warning_dialog(title: str, message: str, parent=None):
    """
    显示警告对话框
    
    Args:
        title: 对话框标题
        message: 警告消息
        parent: 父窗口
    """
    messagebox.showwarning(title, message, parent=parent)


def ask_yes_no(title: str, message: str, parent=None) -> bool:
    """
    显示是/否确认对话框
    
    Args:
        title: 对话框标题
        message: 确认消息
        parent: 父窗口
        
    Returns:
        bool: 用户选择结果
    """
    return messagebox.askyesno(title, message, parent=parent)


def open_file_location(file_path: str):
    """
    在文件管理器中打开文件位置
    
    Args:
        file_path: 文件路径
    """
    try:
        if platform.system() == "Windows":
            subprocess.run(['explorer', '/select,', file_path])
        elif platform.system() == "Darwin":  # macOS
            subprocess.run(['open', '-R', file_path])
        else:  # Linux
            subprocess.run(['xdg-open', os.path.dirname(file_path)])
    except Exception as e:
        show_error_dialog("错误", f"无法打开文件位置：{str(e)}")


def get_system_info() -> dict:
    """
    获取系统信息
    
    Returns:
        dict: 系统信息字典
    """
    return {
        'platform': platform.system(),
        'platform_release': platform.release(),
        'platform_version': platform.version(),
        'architecture': platform.machine(),
        'processor': platform.processor(),
        'python_version': platform.python_version(),
        'python_implementation': platform.python_implementation()
    }


def safe_filename(filename: str) -> str:
    """
    生成安全的文件名（移除非法字符）
    
    Args:
        filename: 原始文件名
        
    Returns:
        str: 安全的文件名
    """
    # Windows文件名非法字符
    illegal_chars = '<>:"/\\|?*'
    
    safe_name = filename
    for char in illegal_chars:
        safe_name = safe_name.replace(char, '_')
    
    # 移除前后空格和点
    safe_name = safe_name.strip(' .')
    
    # 确保文件名不为空
    if not safe_name:
        safe_name = "untitled"
    
    return safe_name


def truncate_text(text: str, max_length: int, suffix: str = "...") -> str:
    """
    截断文本到指定长度
    
    Args:
        text: 原始文本
        max_length: 最大长度
        suffix: 截断后缀
        
    Returns:
        str: 截断后的文本
    """
    if len(text) <= max_length:
        return text
    
    return text[:max_length - len(suffix)] + suffix


def parse_file_list_from_string(file_string: str) -> List[str]:
    """
    从字符串解析文件列表（用于拖拽功能）
    
    Args:
        file_string: 文件路径字符串
        
    Returns:
        List[str]: 文件路径列表
    """
    files = []
    
    # 处理多种可能的分隔符
    if '\n' in file_string:
        # 换行分隔
        raw_files = file_string.split('\n')
    elif '\r' in file_string:
        # 回车分隔
        raw_files = file_string.split('\r')
    else:
        # 空格分隔（可能有引号包围）
        raw_files = []
        current_file = ""
        in_quotes = False
        
        for char in file_string:
            if char == '"':
                in_quotes = not in_quotes
            elif char == ' ' and not in_quotes:
                if current_file.strip():
                    raw_files.append(current_file.strip())
                current_file = ""
            else:
                current_file += char
        
        if current_file.strip():
            raw_files.append(current_file.strip())
    
    # 清理和验证文件路径
    for file_path in raw_files:
        file_path = file_path.strip().strip('"\'')
        if file_path and os.path.exists(file_path):
            files.append(file_path)
    
    return files
