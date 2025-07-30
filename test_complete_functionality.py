#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试完整功能
"""

import os
import sys
import tempfile
import shutil

# 添加项目路径
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'music_unlock_gui'))

from core.processor import FileProcessor

def test_complete_functionality():
    """测试完整功能"""
    try:
        # 初始化处理器
        um_exe_path = os.path.join(os.path.dirname(__file__), 'um.exe')
        processor = FileProcessor(um_exe_path)
        
        print("=== 完整功能测试 ===")
        
        # 测试文件
        test_file = os.path.join("testdata", "听妈妈的话 (Live) - 周杰伦 _ 潘玮柏 _ 张学友.mflac")
        
        if not os.path.exists(test_file):
            print(f"测试文件不存在: {test_file}")
            return False
        
        print(f"测试文件: {test_file}")
        print(f"文件大小: {os.path.getsize(test_file)} bytes")
        
        # 测试格式检查
        is_supported = processor.is_supported_file(test_file)
        print(f"格式支持检查: {'✓ 支持' if is_supported else '✗ 不支持'}")
        
        if not is_supported:
            print("文件格式不支持，跳过处理测试")
            return False
        
        # 测试1：使用源目录模式
        print(f"\n测试1 - 源目录模式处理:")
        success, message = processor.process_file(
            test_file, 
            use_source_dir=True
        )
        print(f"处理结果: {'✓ 成功' if success else '✗ 失败'}")
        print(f"消息: {message}")
        
        # 检查输出文件是否在源目录
        source_dir = os.path.dirname(test_file)
        expected_output = os.path.join(source_dir, "听妈妈的话 (Live) - 周杰伦 _ 潘玮柏 _ 张学友.flac")
        if os.path.exists(expected_output):
            print(f"✓ 输出文件已生成: {expected_output}")
            print(f"  文件大小: {os.path.getsize(expected_output)} bytes")
        else:
            print(f"✗ 输出文件未找到: {expected_output}")
        
        # 测试2：使用自定义目录模式
        custom_output_dir = "output"
        os.makedirs(custom_output_dir, exist_ok=True)
        
        print(f"\n测试2 - 自定义目录模式处理:")
        print(f"自定义输出目录: {custom_output_dir}")
        
        success, message = processor.process_file(
            test_file, 
            output_dir=custom_output_dir,
            use_source_dir=False
        )
        print(f"处理结果: {'✓ 成功' if success else '✗ 失败'}")
        print(f"消息: {message}")
        
        # 检查输出文件是否在自定义目录
        expected_output = os.path.join(custom_output_dir, "听妈妈的话 (Live) - 周杰伦 _ 潘玮柏 _ 张学友.flac")
        if os.path.exists(expected_output):
            print(f"✓ 输出文件已生成: {expected_output}")
            print(f"  文件大小: {os.path.getsize(expected_output)} bytes")
        else:
            print(f"✗ 输出文件未找到: {expected_output}")
        
        return True
        
    except Exception as e:
        print(f"测试失败: {e}")
        import traceback
        traceback.print_exc()
        return False

if __name__ == "__main__":
    success = test_complete_functionality()
    sys.exit(0 if success else 1)
