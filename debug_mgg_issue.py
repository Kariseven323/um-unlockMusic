#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
MGG文件处理问题诊断脚本
用于对比直接调用和GUI调用um.exe的差异，找出mgg文件处理失败的原因
"""

import os
import sys
import subprocess
import platform
import json
import tempfile
import shutil
from pathlib import Path

def get_subprocess_kwargs():
    """获取subprocess调用的参数，在Windows下隐藏控制台窗口"""
    kwargs = {}
    if platform.system() == 'Windows':
        kwargs['creationflags'] = subprocess.CREATE_NO_WINDOW
    return kwargs

def test_simple_call(um_exe_path, mgg_file):
    """测试简单调用方式（模拟拖拽）"""
    print("=" * 60)
    print("测试1: 简单调用方式（模拟拖拽到um.exe）")
    print("=" * 60)
    
    try:
        # 切换到mgg文件所在目录
        original_cwd = os.getcwd()
        mgg_dir = os.path.dirname(os.path.abspath(mgg_file))
        mgg_filename = os.path.basename(mgg_file)
        
        print(f"原始工作目录: {original_cwd}")
        print(f"切换到目录: {mgg_dir}")
        print(f"文件名: {mgg_filename}")
        
        os.chdir(mgg_dir)
        
        # 简单调用
        cmd = [um_exe_path, mgg_filename]
        print(f"执行命令: {' '.join(cmd)}")
        print(f"工作目录: {os.getcwd()}")
        
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            encoding='utf-8',
            errors='ignore',
            timeout=60,
            **get_subprocess_kwargs()
        )
        
        print(f"返回码: {result.returncode}")
        print(f"标准输出:\n{result.stdout}")
        print(f"错误输出:\n{result.stderr}")
        
        # 检查输出文件
        expected_output = mgg_filename.replace('.mgg', '.ogg')
        if os.path.exists(expected_output):
            print(f"✅ 成功生成输出文件: {expected_output}")
            file_size = os.path.getsize(expected_output)
            print(f"输出文件大小: {file_size} 字节")
        else:
            print(f"❌ 未找到预期输出文件: {expected_output}")
        
        os.chdir(original_cwd)
        return result.returncode == 0
        
    except Exception as e:
        print(f"❌ 简单调用异常: {e}")
        os.chdir(original_cwd)
        return False

def test_gui_call(um_exe_path, mgg_file, output_dir):
    """测试GUI调用方式"""
    print("\n" + "=" * 60)
    print("测试2: GUI调用方式（复杂参数）")
    print("=" * 60)
    
    try:
        # 确保输出目录存在
        os.makedirs(output_dir, exist_ok=True)
        
        # GUI调用方式
        cmd = [
            um_exe_path,
            '-i', mgg_file,
            '-o', output_dir,
            '--naming-format', 'auto',
            '--overwrite',
            '--verbose'
        ]
        
        print(f"执行命令: {' '.join(cmd)}")
        print(f"工作目录: {os.getcwd()}")
        print(f"输入文件: {mgg_file}")
        print(f"输出目录: {output_dir}")
        
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            encoding='utf-8',
            errors='ignore',
            timeout=60,
            **get_subprocess_kwargs()
        )
        
        print(f"返回码: {result.returncode}")
        print(f"标准输出:\n{result.stdout}")
        print(f"错误输出:\n{result.stderr}")
        
        # 检查输出文件
        mgg_basename = os.path.splitext(os.path.basename(mgg_file))[0]
        expected_output = os.path.join(output_dir, f"{mgg_basename}.ogg")
        if os.path.exists(expected_output):
            print(f"✅ 成功生成输出文件: {expected_output}")
            file_size = os.path.getsize(expected_output)
            print(f"输出文件大小: {file_size} 字节")
        else:
            print(f"❌ 未找到预期输出文件: {expected_output}")
            # 列出输出目录的所有文件
            if os.path.exists(output_dir):
                files = os.listdir(output_dir)
                print(f"输出目录中的文件: {files}")
        
        return result.returncode == 0
        
    except Exception as e:
        print(f"❌ GUI调用异常: {e}")
        return False

def test_path_encoding(um_exe_path, mgg_file):
    """测试路径编码问题"""
    print("\n" + "=" * 60)
    print("测试3: 路径编码测试")
    print("=" * 60)
    
    print(f"文件路径: {mgg_file}")
    print(f"路径编码: {mgg_file.encode('utf-8')}")
    print(f"路径存在: {os.path.exists(mgg_file)}")
    print(f"绝对路径: {os.path.abspath(mgg_file)}")
    
    # 测试不同编码方式
    encodings = ['utf-8', 'gbk', 'cp936']
    for encoding in encodings:
        try:
            encoded_path = mgg_file.encode(encoding).decode(encoding)
            print(f"{encoding} 编码测试: {encoded_path == mgg_file}")
        except Exception as e:
            print(f"{encoding} 编码失败: {e}")

def test_working_directory(um_exe_path, mgg_file, output_dir):
    """测试工作目录影响"""
    print("\n" + "=" * 60)
    print("测试4: 工作目录影响测试")
    print("=" * 60)
    
    original_cwd = os.getcwd()
    
    # 测试不同工作目录
    test_dirs = [
        os.path.dirname(um_exe_path),  # um.exe所在目录
        os.path.dirname(mgg_file),     # mgg文件所在目录
        output_dir,                    # 输出目录
        original_cwd                   # 原始工作目录
    ]
    
    for i, test_dir in enumerate(test_dirs, 1):
        print(f"\n测试4.{i}: 工作目录 = {test_dir}")
        try:
            os.chdir(test_dir)
            
            cmd = [
                um_exe_path,
                '-i', mgg_file,
                '-o', output_dir,
                '--naming-format', 'auto',
                '--overwrite',
                '--verbose'
            ]
            
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                encoding='utf-8',
                errors='ignore',
                timeout=30,
                **get_subprocess_kwargs()
            )
            
            print(f"返回码: {result.returncode}")
            if result.returncode != 0:
                print(f"错误输出: {result.stderr}")
            else:
                print("✅ 成功")
                
        except Exception as e:
            print(f"❌ 异常: {e}")
        finally:
            os.chdir(original_cwd)

def check_um_exe_info(um_exe_path):
    """检查um.exe基本信息"""
    print("=" * 60)
    print("um.exe 基本信息检查")
    print("=" * 60)
    
    print(f"um.exe路径: {um_exe_path}")
    print(f"文件存在: {os.path.exists(um_exe_path)}")
    
    if os.path.exists(um_exe_path):
        file_size = os.path.getsize(um_exe_path)
        print(f"文件大小: {file_size} 字节")
        
        # 获取版本信息
        try:
            result = subprocess.run(
                [um_exe_path, '--version'],
                capture_output=True,
                text=True,
                encoding='utf-8',
                errors='ignore',
                timeout=10,
                **get_subprocess_kwargs()
            )
            print(f"版本信息: {result.stdout.strip()}")
        except Exception as e:
            print(f"获取版本信息失败: {e}")
        
        # 获取支持格式
        try:
            result = subprocess.run(
                [um_exe_path, '--supported-ext'],
                capture_output=True,
                text=True,
                encoding='utf-8',
                errors='ignore',
                timeout=10,
                **get_subprocess_kwargs()
            )
            if '.mgg' in result.stdout:
                print("✅ 支持mgg格式")
            else:
                print("❌ 不支持mgg格式")
                print(f"支持的格式:\n{result.stdout}")
        except Exception as e:
            print(f"获取支持格式失败: {e}")

def main():
    """主函数"""
    print("MGG文件处理问题诊断脚本")
    print("=" * 60)
    
    # 检查参数
    if len(sys.argv) < 3:
        print("用法: python debug_mgg_issue.py <um.exe路径> <mgg文件路径>")
        print("示例: python debug_mgg_issue.py ./um.exe ./test.mgg")
        sys.exit(1)
    
    um_exe_path = sys.argv[1]
    mgg_file = sys.argv[2]
    
    # 创建临时输出目录
    output_dir = tempfile.mkdtemp(prefix="mgg_test_")
    print(f"临时输出目录: {output_dir}")
    
    try:
        # 基本信息检查
        check_um_exe_info(um_exe_path)
        
        # 路径编码测试
        test_path_encoding(um_exe_path, mgg_file)
        
        # 简单调用测试
        simple_success = test_simple_call(um_exe_path, mgg_file)
        
        # GUI调用测试
        gui_success = test_gui_call(um_exe_path, mgg_file, output_dir)
        
        # 工作目录测试
        test_working_directory(um_exe_path, mgg_file, output_dir)
        
        # 总结
        print("\n" + "=" * 60)
        print("测试总结")
        print("=" * 60)
        print(f"简单调用（拖拽模拟）: {'✅ 成功' if simple_success else '❌ 失败'}")
        print(f"GUI调用（复杂参数）: {'✅ 成功' if gui_success else '❌ 失败'}")
        
        if simple_success and not gui_success:
            print("\n🔍 问题确认：GUI调用方式存在问题")
            print("建议检查：")
            print("1. 工作目录设置")
            print("2. 参数传递方式")
            print("3. 路径编码处理")
            print("4. 输出目录权限")
        elif not simple_success and not gui_success:
            print("\n🔍 问题确认：um.exe本身无法处理该mgg文件")
        elif simple_success and gui_success:
            print("\n✅ 两种方式都成功，可能是间歇性问题")
        
    finally:
        # 清理临时目录
        try:
            shutil.rmtree(output_dir)
            print(f"\n清理临时目录: {output_dir}")
        except Exception as e:
            print(f"清理临时目录失败: {e}")

if __name__ == "__main__":
    main()
