#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试批处理模式功能
"""

import json
import subprocess
import os
import time

def test_batch_mode():
    """测试批处理模式"""
    print("=== 批处理模式测试 ===")
    
    # 检查um.exe是否存在
    um_exe_path = "um.exe"
    if not os.path.exists(um_exe_path):
        print(f"错误: {um_exe_path} 不存在")
        return False
    
    # 准备测试数据
    test_request = {
        "files": [
            {
                "input_path": "testdata/听妈妈的话 (Live) - 周杰伦 _ 潘玮柏 _ 张学友.mflac",
                "output_path": "output"
            }
        ],
        "options": {
            "remove_source": False,
            "update_metadata": True,
            "overwrite_output": True,
            "skip_noop": True
        }
    }
    
    # 检查测试文件是否存在
    test_file = test_request["files"][0]["input_path"]
    if not os.path.exists(test_file):
        print(f"警告: 测试文件 {test_file} 不存在，使用模拟数据")
        # 使用一个不存在的文件来测试错误处理
        test_request["files"][0]["input_path"] = "nonexistent.mflac"
    
    print(f"测试请求: {json.dumps(test_request, indent=2, ensure_ascii=False)}")
    
    try:
        # 调用批处理模式
        start_time = time.time()
        
        process = subprocess.Popen(
            [um_exe_path, "--batch"],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            encoding='utf-8'
        )
        
        # 发送JSON请求
        stdout, stderr = process.communicate(
            input=json.dumps(test_request),
            timeout=30
        )
        
        end_time = time.time()
        
        print(f"执行时间: {end_time - start_time:.2f}秒")
        print(f"返回码: {process.returncode}")
        
        if stderr:
            print(f"错误输出: {stderr}")
        
        if stdout:
            print(f"标准输出: {stdout}")
            
            # 尝试解析响应
            try:
                response = json.loads(stdout)
                print("批处理响应解析成功:")
                print(f"  总文件数: {response.get('total_files', 0)}")
                print(f"  成功数: {response.get('success_count', 0)}")
                print(f"  失败数: {response.get('failed_count', 0)}")
                print(f"  总耗时: {response.get('total_time_ms', 0)}ms")
                
                for i, result in enumerate(response.get('results', [])):
                    print(f"  文件{i+1}: {result.get('input_path', 'unknown')}")
                    print(f"    成功: {result.get('success', False)}")
                    if result.get('error'):
                        print(f"    错误: {result.get('error')}")
                    print(f"    耗时: {result.get('process_time_ms', 0)}ms")
                
                return True
                
            except json.JSONDecodeError as e:
                print(f"响应JSON解析失败: {e}")
                return False
        else:
            print("没有输出")
            return False
            
    except subprocess.TimeoutExpired:
        print("执行超时")
        process.kill()
        return False
    except Exception as e:
        print(f"执行异常: {e}")
        return False

def test_help_command():
    """测试help命令是否包含batch选项"""
    print("\n=== Help命令测试 ===")
    
    try:
        result = subprocess.run(
            ["um.exe", "--help"],
            capture_output=True,
            text=True,
            timeout=10
        )
        
        if result.returncode == 0:
            help_text = result.stdout
            if "--batch" in help_text or "batch" in help_text:
                print("✓ Help中包含batch选项")
                return True
            else:
                print("✗ Help中未找到batch选项")
                print("Help输出:")
                print(help_text)
                return False
        else:
            print(f"Help命令执行失败，返回码: {result.returncode}")
            print(f"错误: {result.stderr}")
            return False
            
    except Exception as e:
        print(f"Help命令测试异常: {e}")
        return False

if __name__ == "__main__":
    print("开始测试批处理模式功能...")
    
    # 测试help命令
    help_ok = test_help_command()
    
    # 测试批处理模式
    batch_ok = test_batch_mode()
    
    print(f"\n=== 测试结果 ===")
    print(f"Help命令测试: {'✓ 通过' if help_ok else '✗ 失败'}")
    print(f"批处理模式测试: {'✓ 通过' if batch_ok else '✗ 失败'}")
    
    if help_ok and batch_ok:
        print("所有测试通过！")
    else:
        print("部分测试失败，需要检查实现。")
