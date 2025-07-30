#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试格式支持功能
"""

import os
import sys

# 添加项目路径
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'music_unlock_gui'))

from core.processor import FileProcessor

def test_format_support():
    """测试格式支持功能"""
    try:
        # 初始化处理器
        um_exe_path = os.path.join(os.path.dirname(__file__), 'um.exe')
        processor = FileProcessor(um_exe_path)
        
        print("=== 格式支持测试 ===")
        print(f"支持的格式数量: {len(processor.supported_extensions)}")
        print("\n支持的格式列表:")
        for i, ext in enumerate(sorted(processor.supported_extensions), 1):
            print(f"{i:2d}. {ext}")
        
        # 测试特定格式
        test_files = [
            "test.mflac",
            "test.mgg", 
            "test.mflac0",
            "test.mgg1",
            "test.ncm",
            "test.qmc0",
            "test.unsupported"
        ]
        
        print("\n=== 格式检查测试 ===")
        for test_file in test_files:
            is_supported = processor.is_supported_file(test_file)
            status = "✓ 支持" if is_supported else "✗ 不支持"
            print(f"{test_file:<15} {status}")
            
    except Exception as e:
        print(f"测试失败: {e}")
        return False
    
    return True

if __name__ == "__main__":
    success = test_format_support()
    sys.exit(0 if success else 1)
