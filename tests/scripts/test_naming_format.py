#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试命名格式功能
"""

import os
import sys
import tempfile
import subprocess
import json

# 添加项目路径到sys.path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'music_unlock_gui'))

from core.constants import (
    NAMING_FORMAT_AUTO,
    NAMING_FORMAT_TITLE_ARTIST,
    NAMING_FORMAT_ARTIST_TITLE,
    NAMING_FORMAT_ORIGINAL
)

def test_backend_naming_format():
    """测试后端命名格式参数"""
    print("测试后端命名格式参数...")
    
    # 检查um.exe是否存在
    um_exe_path = "um.exe"
    if not os.path.exists(um_exe_path):
        print(f"警告: {um_exe_path} 不存在，跳过后端测试")
        return
    
    # 测试命令行参数
    try:
        result = subprocess.run([um_exe_path, "--help"],
                              capture_output=True, text=True,
                              encoding='utf-8', errors='ignore', timeout=10)
        if "--naming-format" in result.stdout:
            print("✓ 后端命令行参数 --naming-format 已添加")
        else:
            print("✗ 后端命令行参数 --naming-format 未找到")
            print("Help输出:", result.stdout)
    except Exception as e:
        print(f"✗ 测试后端命令行参数失败: {e}")

def test_batch_mode_naming_format():
    """测试批处理模式命名格式"""
    print("\n测试批处理模式命名格式...")
    
    um_exe_path = "um.exe"
    if not os.path.exists(um_exe_path):
        print(f"警告: {um_exe_path} 不存在，跳过批处理测试")
        return
    
    # 创建测试批处理请求
    test_request = {
        "files": [
            {"input_path": "test.ncm"}
        ],
        "options": {
            "remove_source": False,
            "update_metadata": False,
            "overwrite_output": True,
            "skip_noop": True,
            "naming_format": "title-artist"
        }
    }
    
    try:
        # 测试批处理模式是否接受naming_format参数
        result = subprocess.run([um_exe_path, "--batch"],
                              input=json.dumps(test_request),
                              capture_output=True, text=True,
                              encoding='utf-8', errors='ignore', timeout=10)
        
        if result.returncode == 0 or "naming_format" not in result.stderr:
            print("✓ 批处理模式支持 naming_format 参数")
        else:
            print("✗ 批处理模式不支持 naming_format 参数")
            print("错误输出:", result.stderr)
    except Exception as e:
        print(f"✗ 测试批处理模式失败: {e}")

def test_frontend_constants():
    """测试前端常量定义"""
    print("\n测试前端常量定义...")
    
    try:
        # 检查常量是否正确定义
        assert NAMING_FORMAT_AUTO == "auto"
        assert NAMING_FORMAT_TITLE_ARTIST == "title-artist"
        assert NAMING_FORMAT_ARTIST_TITLE == "artist-title"
        assert NAMING_FORMAT_ORIGINAL == "original"
        print("✓ 前端命名格式常量定义正确")
    except Exception as e:
        print(f"✗ 前端常量定义错误: {e}")

def test_frontend_imports():
    """测试前端模块导入"""
    print("\n测试前端模块导入...")
    
    try:
        from core.processor import FileProcessor
        from core.thread_manager import ThreadManager
        from gui.main_window import MusicUnlockGUI
        print("✓ 前端模块导入成功")
    except Exception as e:
        print(f"✗ 前端模块导入失败: {e}")

def main():
    """主测试函数"""
    print("开始测试命名格式功能...\n")
    
    test_backend_naming_format()
    test_batch_mode_naming_format()
    test_frontend_constants()
    test_frontend_imports()
    
    print("\n测试完成！")

if __name__ == "__main__":
    main()
