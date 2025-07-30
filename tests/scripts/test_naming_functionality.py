#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试命名格式功能的实际效果
"""

import os
import sys
import tempfile
import subprocess
import json
import shutil

def test_naming_format_with_real_file():
    """使用真实文件测试命名格式功能"""
    print("测试命名格式功能的实际效果...")
    
    # 检查是否有测试文件
    test_file = "test.mflac"
    if not os.path.exists(test_file):
        print(f"警告: 测试文件 {test_file} 不存在，跳过实际文件测试")
        return
    
    um_exe_path = "um.exe"
    if not os.path.exists(um_exe_path):
        print(f"警告: {um_exe_path} 不存在，跳过实际文件测试")
        return
    
    # 创建临时输出目录
    with tempfile.TemporaryDirectory() as temp_dir:
        print(f"使用临时目录: {temp_dir}")
        
        # 测试不同的命名格式
        formats_to_test = [
            ("auto", "智能识别"),
            ("title-artist", "歌曲名-歌手名"),
            ("artist-title", "歌手名-歌曲名"),
            ("original", "原文件名")
        ]
        
        for format_name, format_desc in formats_to_test:
            print(f"\n测试命名格式: {format_desc} ({format_name})")
            
            # 为每种格式创建子目录
            format_dir = os.path.join(temp_dir, format_name)
            os.makedirs(format_dir, exist_ok=True)
            
            try:
                # 使用命令行参数测试
                cmd = [
                    um_exe_path,
                    "-i", test_file,
                    "-o", format_dir,
                    "--naming-format", format_name,
                    "--overwrite"
                ]
                
                print(f"执行命令: {' '.join(cmd)}")
                result = subprocess.run(cmd, 
                                      capture_output=True, text=True,
                                      encoding='utf-8', errors='ignore', timeout=30)
                
                if result.returncode == 0:
                    # 检查输出文件
                    output_files = [f for f in os.listdir(format_dir) if f.endswith(('.mp3', '.flac', '.ogg'))]
                    if output_files:
                        print(f"✓ 成功生成文件: {output_files[0]}")
                    else:
                        print("✗ 未找到输出文件")
                else:
                    print(f"✗ 命令执行失败: {result.stderr}")
                    
            except Exception as e:
                print(f"✗ 测试失败: {e}")

def test_batch_mode_naming():
    """测试批处理模式的命名格式"""
    print("\n测试批处理模式命名格式...")
    
    test_file = "test.mflac"
    if not os.path.exists(test_file):
        print(f"警告: 测试文件 {test_file} 不存在，跳过批处理测试")
        return
    
    um_exe_path = "um.exe"
    if not os.path.exists(um_exe_path):
        print(f"警告: {um_exe_path} 不存在，跳过批处理测试")
        return
    
    with tempfile.TemporaryDirectory() as temp_dir:
        print(f"使用临时目录: {temp_dir}")
        
        # 创建批处理请求
        batch_request = {
            "files": [
                {"input_path": test_file}
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
            result = subprocess.run([um_exe_path, "--batch"], 
                                  input=json.dumps(batch_request),
                                  capture_output=True, text=True,
                                  encoding='utf-8', errors='ignore', timeout=30)
            
            if result.returncode == 0:
                print("✓ 批处理模式执行成功")
                try:
                    response = json.loads(result.stdout)
                    print(f"处理结果: {response}")
                except:
                    print("响应解析失败，但命令执行成功")
            else:
                print(f"✗ 批处理模式失败: {result.stderr}")
                
        except Exception as e:
            print(f"✗ 批处理测试失败: {e}")

def test_frontend_integration():
    """测试前端集成"""
    print("\n测试前端集成...")
    
    try:
        # 添加项目路径
        sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'music_unlock_gui'))
        
        from core.constants import NAMING_FORMAT_AUTO, NAMING_FORMAT_LABELS
        from core.processor import FileProcessor
        
        # 创建处理器实例
        processor = FileProcessor("um.exe")
        
        # 测试方法签名
        import inspect
        sig = inspect.signature(processor.process_file)
        if 'naming_format' in sig.parameters:
            print("✓ FileProcessor.process_file 支持 naming_format 参数")
        else:
            print("✗ FileProcessor.process_file 缺少 naming_format 参数")
            
        sig_batch = inspect.signature(processor.process_files_batch)
        if 'naming_format' in sig_batch.parameters:
            print("✓ FileProcessor.process_files_batch 支持 naming_format 参数")
        else:
            print("✗ FileProcessor.process_files_batch 缺少 naming_format 参数")
            
        print(f"✓ 命名格式选项: {list(NAMING_FORMAT_LABELS.keys())}")
        
    except Exception as e:
        print(f"✗ 前端集成测试失败: {e}")

def main():
    """主测试函数"""
    print("开始测试命名格式功能的实际效果...\n")
    
    test_naming_format_with_real_file()
    test_batch_mode_naming()
    test_frontend_integration()
    
    print("\n功能测试完成！")

if __name__ == "__main__":
    main()
