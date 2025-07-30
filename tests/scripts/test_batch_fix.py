#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试批处理修复效果
"""

import json
import subprocess
import os

def test_batch_fix():
    """测试批处理修复是否生效"""
    print("=== 测试批处理修复效果 ===")
    
    # 检查um.exe是否存在
    um_exe_path = "um.exe"
    if not os.path.exists(um_exe_path):
        print(f"✗ {um_exe_path} 不存在")
        return False
    
    # 准备测试数据
    test_request = {
        "files": [
            {"input_path": "nonexistent1.mflac"},
            {"input_path": "nonexistent2.mflac"}
        ],
        "options": {
            "remove_source": False,
            "update_metadata": True,
            "overwrite_output": True,
            "skip_noop": True
        }
    }
    
    print(f"测试请求: {json.dumps(test_request, indent=2, ensure_ascii=False)}")
    
    try:
        # 调用批处理模式
        cmd = [um_exe_path, "--batch"]
        
        result = subprocess.run(
            cmd,
            input=json.dumps(test_request),
            capture_output=True,
            text=True,
            encoding='utf-8',
            timeout=30
        )
        
        print(f"返回码: {result.returncode}")
        print(f"标准输出长度: {len(result.stdout)} 字符")
        print(f"标准错误长度: {len(result.stderr)} 字符")
        
        if result.stderr:
            print(f"标准错误内容（前200字符）: {result.stderr[:200]}")
        
        if result.stdout:
            print(f"标准输出内容: {result.stdout}")
            
            # 尝试解析JSON
            try:
                response = json.loads(result.stdout)
                print("✓ JSON解析成功！")
                print(f"  总文件数: {response.get('total_files', 0)}")
                print(f"  成功数: {response.get('success_count', 0)}")
                print(f"  失败数: {response.get('failed_count', 0)}")
                print(f"  总耗时: {response.get('total_time_ms', 0)}ms")
                
                return True
                
            except json.JSONDecodeError as e:
                print(f"✗ JSON解析仍然失败: {e}")
                print(f"输出内容: '{result.stdout}'")
                return False
        else:
            print("✗ 没有标准输出")
            return False
            
    except subprocess.TimeoutExpired:
        print("✗ 执行超时")
        return False
    except Exception as e:
        print(f"✗ 执行异常: {e}")
        return False

if __name__ == "__main__":
    print("开始测试批处理修复效果...")
    
    success = test_batch_fix()
    
    print(f"\n=== 测试结果 ===")
    if success:
        print("🎉 批处理修复成功！JSON解析正常。")
    else:
        print("❌ 批处理修复失败，仍有问题。")
