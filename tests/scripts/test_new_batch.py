#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试新编译的um_new.exe的批处理功能
"""

import json
import subprocess
import os
import tempfile

def test_batch_mode_basic():
    """测试基本的批处理模式"""
    print("=== 测试基本批处理模式 ===")
    
    # 创建简单的测试请求（使用不存在的文件测试错误处理）
    test_request = {
        "files": [
            {"input_path": "nonexistent.mflac"}
        ],
        "options": {
            "remove_source": False,
            "update_metadata": False,
            "overwrite_output": True,
            "skip_noop": True
        }
    }
    
    try:
        result = subprocess.run(
            ["./um_new.exe", "--batch"],
            input=json.dumps(test_request),
            capture_output=True,
            text=True,
            encoding='utf-8',
            timeout=30
        )
        
        print(f"返回码: {result.returncode}")
        print(f"标准输出: {result.stdout}")
        if result.stderr:
            print(f"错误输出: {result.stderr}")
        
        if result.stdout:
            try:
                response = json.loads(result.stdout)
                print("✓ 批处理响应解析成功")
                print(f"  总文件数: {response.get('total_files', 0)}")
                print(f"  成功数: {response.get('success_count', 0)}")
                print(f"  失败数: {response.get('failed_count', 0)}")
                print(f"  处理时间: {response.get('total_time_ms', 0)}ms")
                return True
            except json.JSONDecodeError as e:
                print(f"✗ 响应JSON解析失败: {e}")
                return False
        else:
            print("✗ 没有输出")
            return False
            
    except subprocess.TimeoutExpired:
        print("✗ 执行超时")
        return False
    except Exception as e:
        print(f"✗ 测试异常: {e}")
        return False

def test_batch_mode_with_real_file():
    """使用真实文件测试批处理模式"""
    print("\n=== 测试真实文件批处理 ===")
    
    # 检查是否有测试文件
    test_file = "testdata/听妈妈的话 (Live) - 周杰伦 _ 潘玮柏 _ 张学友.mflac"
    if not os.path.exists(test_file):
        print(f"✗ 测试文件不存在: {test_file}")
        return False
    
    # 创建临时输出目录
    output_dir = tempfile.mkdtemp(prefix="um_test_")
    
    test_request = {
        "files": [
            {"input_path": test_file, "output_path": output_dir}
        ],
        "options": {
            "remove_source": False,
            "update_metadata": False,
            "overwrite_output": True,
            "skip_noop": True
        }
    }
    
    try:
        result = subprocess.run(
            ["./um_new.exe", "--batch"],
            input=json.dumps(test_request),
            capture_output=True,
            text=True,
            encoding='utf-8',
            timeout=60
        )
        
        print(f"返回码: {result.returncode}")
        print(f"标准输出: {result.stdout}")
        if result.stderr:
            print(f"错误输出: {result.stderr}")
        
        if result.stdout:
            try:
                response = json.loads(result.stdout)
                print("✓ 批处理响应解析成功")
                print(f"  总文件数: {response.get('total_files', 0)}")
                print(f"  成功数: {response.get('success_count', 0)}")
                print(f"  失败数: {response.get('failed_count', 0)}")
                print(f"  处理时间: {response.get('total_time_ms', 0)}ms")
                
                # 检查结果
                if response.get('success_count', 0) > 0:
                    print("✓ 文件处理成功")
                    # 检查输出文件
                    output_files = os.listdir(output_dir)
                    print(f"  输出文件: {output_files}")
                    return True
                else:
                    print("✗ 文件处理失败")
                    if response.get('results'):
                        for result in response['results']:
                            if not result.get('success', False):
                                print(f"    错误: {result.get('error', 'Unknown error')}")
                    return False
                    
            except json.JSONDecodeError as e:
                print(f"✗ 响应JSON解析失败: {e}")
                return False
        else:
            print("✗ 没有输出")
            return False
            
    except subprocess.TimeoutExpired:
        print("✗ 执行超时")
        return False
    except Exception as e:
        print(f"✗ 测试异常: {e}")
        return False
    finally:
        # 清理临时目录
        import shutil
        try:
            shutil.rmtree(output_dir)
        except:
            pass

def main():
    """主函数"""
    print("测试新编译的um_new.exe批处理功能")
    print("=" * 50)
    
    # 检查um_new.exe是否存在
    if not os.path.exists("um_new.exe"):
        print("错误: um_new.exe 不存在")
        return False
    
    # 测试基本批处理模式
    basic_ok = test_batch_mode_basic()
    
    # 测试真实文件处理
    real_file_ok = test_batch_mode_with_real_file()
    
    print(f"\n=== 测试结果 ===")
    print(f"基本批处理测试: {'✓' if basic_ok else '✗'}")
    print(f"真实文件测试: {'✓' if real_file_ok else '✗'}")
    
    if basic_ok and real_file_ok:
        print("所有测试通过！批处理模式工作正常。")
        return True
    else:
        print("部分测试失败。")
        return False

if __name__ == "__main__":
    main()
