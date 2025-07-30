#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试输出路径功能
"""

import os
import sys
import tempfile
import shutil

# 添加项目路径
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'music_unlock_gui'))

from core.processor import FileProcessor

def test_output_path_logic():
    """测试输出路径逻辑"""
    try:
        # 初始化处理器
        um_exe_path = os.path.join(os.path.dirname(__file__), 'um.exe')
        processor = FileProcessor(um_exe_path)
        
        print("=== 输出路径逻辑测试 ===")
        
        # 创建测试文件路径
        test_file = os.path.join("testdata", "听妈妈的话 (Live) - 周杰伦 _ 潘玮柏 _ 张学友.mflac")
        custom_output = os.path.join(os.getcwd(), "output")
        
        # 测试1：使用源文件目录
        result_dir = processor._determine_output_dir(test_file, None, use_source_dir=True)
        expected_dir = os.path.dirname(os.path.abspath(test_file))
        print(f"测试1 - 使用源文件目录:")
        print(f"  输入文件: {test_file}")
        print(f"  预期输出: {expected_dir}")
        print(f"  实际输出: {result_dir}")
        print(f"  结果: {'✓ 通过' if result_dir == expected_dir else '✗ 失败'}")
        
        # 测试2：使用自定义目录
        result_dir = processor._determine_output_dir(test_file, custom_output, use_source_dir=False)
        print(f"\n测试2 - 使用自定义目录:")
        print(f"  输入文件: {test_file}")
        print(f"  自定义目录: {custom_output}")
        print(f"  实际输出: {result_dir}")
        print(f"  结果: {'✓ 通过' if result_dir == custom_output else '✗ 失败'}")
        
        # 测试3：无自定义目录，强制使用源目录
        result_dir = processor._determine_output_dir(test_file, None, use_source_dir=True)
        expected_dir = os.path.dirname(os.path.abspath(test_file))
        print(f"\n测试3 - 无自定义目录，强制使用源目录:")
        print(f"  输入文件: {test_file}")
        print(f"  预期输出: {expected_dir}")
        print(f"  实际输出: {result_dir}")
        print(f"  结果: {'✓ 通过' if result_dir == expected_dir else '✗ 失败'}")
        
        # 测试4：多个不同目录的文件
        test_files = [
            os.path.join("testdata", "file1.mflac"),
            os.path.join("output", "file2.mgg"),
            os.path.join("music_unlock_gui", "file3.ncm")
        ]
        
        print(f"\n测试4 - 多个不同目录的文件:")
        for test_file in test_files:
            result_dir = processor._determine_output_dir(test_file, None, use_source_dir=True)
            expected_dir = os.path.dirname(os.path.abspath(test_file))
            print(f"  文件: {test_file}")
            print(f"  输出目录: {result_dir}")
            print(f"  正确性: {'✓' if result_dir == expected_dir else '✗'}")
            
    except Exception as e:
        print(f"测试失败: {e}")
        return False
    
    return True

if __name__ == "__main__":
    success = test_output_path_logic()
    sys.exit(0 if success else 1)
