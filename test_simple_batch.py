#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
简单的批处理模式测试
"""

import json
import subprocess
import os

def test_batch_help():
    """测试批处理帮助信息"""
    print("=== 测试批处理帮助信息 ===")
    
    try:
        result = subprocess.run(
            ["um.exe", "--help"],
            capture_output=True,
            text=True,
            timeout=10
        )
        
        if result.returncode == 0:
            help_text = result.stdout
            if "batch" in help_text.lower():
                print("✓ 帮助信息中包含batch选项")
                print("相关内容:")
                for line in help_text.split('\n'):
                    if 'batch' in line.lower():
                        print(f"  {line}")
                return True
            else:
                print("✗ 帮助信息中未找到batch选项")
                print("完整帮助信息:")
                print(help_text)
                return False
        else:
            print(f"✗ 获取帮助信息失败，返回码: {result.returncode}")
            print(f"错误: {result.stderr}")
            return False
            
    except Exception as e:
        print(f"✗ 测试异常: {e}")
        return False

def test_batch_mode():
    """测试批处理模式"""
    print("\n=== 测试批处理模式 ===")
    
    # 创建简单的测试请求
    test_request = {
        "files": [
            {"input_path": "nonexistent.mflac"}  # 使用不存在的文件测试错误处理
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
            ["um.exe", "--batch"],
            input=json.dumps(test_request),
            capture_output=True,
            text=True,
            timeout=30
        )
        
        print(f"返回码: {result.returncode}")
        print(f"标准输出: {result.stdout}")
        print(f"错误输出: {result.stderr}")
        
        if result.stdout:
            try:
                response = json.loads(result.stdout)
                print("✓ 批处理响应解析成功")
                print(f"  总文件数: {response.get('total_files', 0)}")
                print(f"  成功数: {response.get('success_count', 0)}")
                print(f"  失败数: {response.get('failed_count', 0)}")
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

def main():
    """主函数"""
    print("简单批处理模式测试")
    print("=" * 40)
    
    # 检查um.exe是否存在
    if not os.path.exists("um.exe"):
        print("错误: um.exe 不存在")
        return False
    
    # 测试帮助信息
    help_ok = test_batch_help()
    
    # 测试批处理模式
    batch_ok = test_batch_mode()
    
    print(f"\n=== 测试结果 ===")
    print(f"帮助信息测试: {'✓' if help_ok else '✗'}")
    print(f"批处理模式测试: {'✓' if batch_ok else '✗'}")
    
    if help_ok and batch_ok:
        print("所有测试通过！批处理模式工作正常。")
        return True
    else:
        print("部分测试失败。可能需要重新编译um.exe。")
        return False

if __name__ == "__main__":
    main()
