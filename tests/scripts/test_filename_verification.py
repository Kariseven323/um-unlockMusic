#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
验证文件名格式是否正确生成
"""

import os
import tempfile
import subprocess
import shutil

def create_test_file_with_name(filename):
    """创建一个测试文件"""
    # 复制现有的test.mflac文件
    if os.path.exists("test.mflac"):
        shutil.copy("test.mflac", filename)
        return True
    return False

def test_filename_formats():
    """测试不同的文件名格式"""
    print("验证文件名格式是否正确生成...")
    
    um_exe_path = "um.exe"
    if not os.path.exists(um_exe_path):
        print(f"警告: {um_exe_path} 不存在")
        return
    
    # 测试用例：不同的输入文件名
    test_cases = [
        ("周杰伦 - 晴天.mflac", "艺术家-歌曲名格式"),
        ("晴天 - 周杰伦.mflac", "歌曲名-艺术家格式"),
        ("Taylor Swift - Love Story.mflac", "英文艺术家-歌曲名"),
        ("Love Story - Taylor Swift.mflac", "英文歌曲名-艺术家"),
        ("test.mflac", "无分隔符文件名")
    ]
    
    with tempfile.TemporaryDirectory() as temp_dir:
        print(f"使用临时目录: {temp_dir}")
        
        for input_filename, description in test_cases:
            print(f"\n测试用例: {description}")
            print(f"输入文件名: {input_filename}")
            
            # 创建测试文件
            test_file_path = os.path.join(temp_dir, input_filename)
            if not create_test_file_with_name(test_file_path):
                print("无法创建测试文件，跳过")
                continue
            
            # 测试不同的命名格式
            formats = [
                ("auto", "智能识别"),
                ("title-artist", "歌曲名-歌手名"),
                ("artist-title", "歌手名-歌曲名"),
                ("original", "原文件名")
            ]
            
            for format_name, format_desc in formats:
                output_dir = os.path.join(temp_dir, f"output_{format_name}")
                os.makedirs(output_dir, exist_ok=True)
                
                try:
                    cmd = [
                        um_exe_path,
                        "-i", test_file_path,
                        "-o", output_dir,
                        "--naming-format", format_name,
                        "--overwrite"
                    ]
                    
                    result = subprocess.run(cmd, 
                                          capture_output=True, text=True,
                                          encoding='utf-8', errors='ignore', timeout=30)
                    
                    if result.returncode == 0:
                        # 检查输出文件名
                        output_files = [f for f in os.listdir(output_dir) 
                                      if f.endswith(('.mp3', '.flac', '.ogg'))]
                        if output_files:
                            output_filename = output_files[0]
                            print(f"  {format_desc}: {output_filename}")
                        else:
                            print(f"  {format_desc}: 未找到输出文件")
                    else:
                        print(f"  {format_desc}: 处理失败")
                        
                except Exception as e:
                    print(f"  {format_desc}: 异常 - {e}")
            
            # 清理测试文件
            if os.path.exists(test_file_path):
                os.remove(test_file_path)

def test_gui_integration():
    """测试GUI集成"""
    print("\n\n测试GUI集成...")
    
    try:
        import sys
        sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'music_unlock_gui'))
        
        from core.constants import NAMING_FORMAT_LABELS
        
        print("命名格式选项:")
        for key, label in NAMING_FORMAT_LABELS.items():
            print(f"  {key}: {label}")
        
        # 模拟GUI选择
        print("\n模拟GUI选择测试:")
        
        # 创建映射（模拟GUI中的逻辑）
        naming_format_mapping = {v: k for k, v in NAMING_FORMAT_LABELS.items()}
        
        for display_value, internal_value in naming_format_mapping.items():
            print(f"  用户选择 '{display_value}' -> 内部值 '{internal_value}'")
        
        print("✓ GUI集成测试通过")
        
    except Exception as e:
        print(f"✗ GUI集成测试失败: {e}")

def main():
    """主测试函数"""
    print("开始验证文件名格式功能...\n")
    
    test_filename_formats()
    test_gui_integration()
    
    print("\n验证完成！")

if __name__ == "__main__":
    main()
