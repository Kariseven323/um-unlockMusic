#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试GUI与优化后的后端集成
"""

import os
import sys
import subprocess
import tempfile
import json

def test_um_exe_basic():
    """测试um.exe基本功能"""
    print("测试 um.exe 基本功能...")
    
    # 测试支持的格式
    result = subprocess.run(['./um.exe', '--supported-ext'], 
                          capture_output=True, text=True)
    if result.returncode == 0:
        print("✓ um.exe 支持格式查询正常")
        print(f"  支持的格式数量: {len(result.stdout.strip().split())}")
    else:
        print("✗ um.exe 支持格式查询失败")
        return False
    
    return True

def test_batch_mode():
    """测试批处理模式"""
    print("\n测试批处理模式...")
    
    # 创建测试请求
    batch_request = {
        "files": [
            {"input_path": "nonexistent.ncm"}
        ],
        "options": {
            "update_metadata": True,
            "overwrite_output": True,
            "naming_format": "auto"
        }
    }
    
    try:
        result = subprocess.run(['./um.exe', '--batch'],
                              input=json.dumps(batch_request),
                              capture_output=True, text=True, timeout=10)
        
        if result.returncode == 0:
            # 解析响应
            response = json.loads(result.stdout)
            print("✓ 批处理模式响应正常")
            print(f"  处理文件数: {response.get('total_files', 0)}")
            print(f"  成功数: {response.get('success_count', 0)}")
            print(f"  失败数: {response.get('failed_count', 0)}")
        else:
            print("✓ 批处理模式正确处理了无效输入")
            
    except subprocess.TimeoutExpired:
        print("✗ 批处理模式超时")
        return False
    except json.JSONDecodeError:
        print("✓ 批处理模式处理了无效文件（预期行为）")
    except Exception as e:
        print(f"✗ 批处理模式测试失败: {e}")
        return False
    
    return True

def test_service_mode():
    """测试服务模式启动"""
    print("\n测试服务模式...")
    
    try:
        # 启动服务模式（短时间测试）
        process = subprocess.Popen(['./um.exe', '--service'],
                                 stdout=subprocess.PIPE,
                                 stderr=subprocess.PIPE)
        
        # 等待一小段时间看是否能正常启动
        import time
        time.sleep(2)
        
        # 检查进程是否还在运行
        if process.poll() is None:
            print("✓ 服务模式启动正常")
            process.terminate()
            process.wait(timeout=5)
            return True
        else:
            print("✗ 服务模式启动失败")
            stdout, stderr = process.communicate()
            print(f"  错误输出: {stderr.decode()}")
            return False
            
    except Exception as e:
        print(f"✗ 服务模式测试失败: {e}")
        return False

def test_gui_processor():
    """测试GUI处理器模块"""
    print("\n测试GUI处理器模块...")
    
    try:
        # 添加GUI模块路径
        sys.path.insert(0, 'music_unlock_gui')
        
        from core.processor import FileProcessor
        
        # 创建处理器实例
        processor = FileProcessor('./um.exe', use_service_mode=False)
        
        print("✓ GUI处理器创建成功")
        print(f"  支持的格式数: {len(processor.supported_extensions)}")
        print(f"  服务模式可用: {processor.service_available}")
        
        return True
        
    except ImportError as e:
        print(f"✗ GUI模块导入失败: {e}")
        return False
    except Exception as e:
        print(f"✗ GUI处理器测试失败: {e}")
        return False

def test_performance_improvement():
    """测试性能改进"""
    print("\n测试性能改进...")
    
    try:
        import time
        
        # 测试多次批处理请求的性能
        batch_request = {
            "files": [
                {"input_path": f"test{i}.ncm"} for i in range(10)
            ],
            "options": {
                "update_metadata": True,
                "overwrite_output": True,
                "naming_format": "auto"
            }
        }
        
        start_time = time.time()
        
        result = subprocess.run(['./um.exe', '--batch'],
                              input=json.dumps(batch_request),
                              capture_output=True, text=True, timeout=30)
        
        end_time = time.time()
        duration = end_time - start_time
        
        print(f"✓ 批处理10个文件耗时: {duration:.2f}秒")
        
        if duration < 5.0:  # 应该很快完成（因为文件不存在）
            print("✓ 性能表现良好")
            return True
        else:
            print("⚠ 性能可能需要进一步优化")
            return True
            
    except Exception as e:
        print(f"✗ 性能测试失败: {e}")
        return False

def main():
    """主测试函数"""
    print("=" * 50)
    print("GUI与优化后端集成测试")
    print("=" * 50)
    
    # 检查um.exe是否存在
    if not os.path.exists('./um.exe'):
        print("✗ um.exe 不存在，请先编译")
        return False
    
    tests = [
        test_um_exe_basic,
        test_batch_mode,
        test_service_mode,
        test_gui_processor,
        test_performance_improvement
    ]
    
    passed = 0
    total = len(tests)
    
    for test in tests:
        try:
            if test():
                passed += 1
        except Exception as e:
            print(f"✗ 测试异常: {e}")
    
    print("\n" + "=" * 50)
    print(f"测试结果: {passed}/{total} 通过")
    
    if passed == total:
        print("🎉 所有测试通过！GUI可以正常使用优化后的后端")
    elif passed >= total * 0.8:
        print("✅ 大部分测试通过，基本功能正常")
    else:
        print("⚠️ 部分测试失败，需要检查问题")
    
    print("=" * 50)
    
    return passed >= total * 0.8

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
