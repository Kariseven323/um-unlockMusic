#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
GUI MGG文件处理修复验证脚本
模拟真实的GUI环境来测试mgg文件处理修复
"""

import os
import sys
import tempfile
import shutil
import time
from pathlib import Path

# 添加项目路径到sys.path
project_root = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, project_root)

from music_unlock_gui.core.processor import FileProcessor
from music_unlock_gui.core.thread_manager import ThreadManager

def create_mock_mgg_file():
    """创建一个模拟的mgg文件用于测试"""
    # 使用现有的mflac文件作为基础，重命名为mgg
    source_file = "./tests/data/test.mflac"
    if not os.path.exists(source_file):
        print(f"❌ 源文件不存在: {source_file}")
        return None
    
    # 创建临时目录
    temp_dir = tempfile.mkdtemp(prefix="mgg_test_")
    mock_mgg_file = os.path.join(temp_dir, "test_mock.mgg")
    
    # 复制文件并重命名
    shutil.copy2(source_file, mock_mgg_file)
    print(f"✅ 创建模拟mgg文件: {mock_mgg_file}")
    
    return mock_mgg_file, temp_dir

def test_processor_direct():
    """直接测试FileProcessor"""
    print("=" * 60)
    print("测试1: 直接FileProcessor测试")
    print("=" * 60)
    
    mock_mgg_file, temp_dir = create_mock_mgg_file()
    if not mock_mgg_file:
        return False
    
    try:
        # 创建输出目录
        output_dir = tempfile.mkdtemp(prefix="mgg_output_")
        
        # 创建处理器
        processor = FileProcessor("./um.exe", use_service_mode=False)
        
        # 测试处理
        print(f"处理文件: {mock_mgg_file}")
        print(f"输出目录: {output_dir}")
        
        success, message = processor.process_file(
            mock_mgg_file,
            output_dir=output_dir,
            use_source_dir=False,
            naming_format="auto"
        )
        
        if success:
            print(f"✅ 处理成功: {message}")
            
            # 检查输出文件
            expected_output = os.path.join(output_dir, "test_mock.ogg")
            if os.path.exists(expected_output):
                file_size = os.path.getsize(expected_output)
                print(f"✅ 输出文件存在: {expected_output} ({file_size} 字节)")
                return True
            else:
                print(f"❌ 输出文件不存在: {expected_output}")
                files = os.listdir(output_dir)
                print(f"输出目录文件: {files}")
                return False
        else:
            print(f"❌ 处理失败: {message}")
            return False
            
    except Exception as e:
        print(f"❌ 测试异常: {e}")
        import traceback
        traceback.print_exc()
        return False
    finally:
        # 清理
        if temp_dir and os.path.exists(temp_dir):
            shutil.rmtree(temp_dir)
        if 'output_dir' in locals() and os.path.exists(output_dir):
            shutil.rmtree(output_dir)

def test_thread_manager():
    """测试ThreadManager（模拟GUI调用）"""
    print("\n" + "=" * 60)
    print("测试2: ThreadManager测试（模拟GUI）")
    print("=" * 60)
    
    mock_mgg_file, temp_dir = create_mock_mgg_file()
    if not mock_mgg_file:
        return False
    
    try:
        # 创建输出目录
        output_dir = tempfile.mkdtemp(prefix="mgg_gui_output_")
        
        # 创建线程管理器
        thread_manager = ThreadManager("./um.exe")
        
        # 模拟GUI的进度回调
        progress_updates = []
        def progress_callback(progress):
            progress_updates.append(progress)
            print(f"进度: {progress}%")
        
        # 模拟GUI的完成回调
        completion_result = {}
        def completion_callback(success, message):
            completion_result['success'] = success
            completion_result['message'] = message
            print(f"完成回调: 成功={success}, 消息={message}")
        
        print(f"处理文件: {mock_mgg_file}")
        print(f"输出目录: {output_dir}")
        
        # 启动处理
        thread_manager.process_file(
            mock_mgg_file,
            output_dir=output_dir,
            use_source_dir=False,
            naming_format="auto",
            progress_callback=progress_callback,
            completion_callback=completion_callback
        )
        
        # 等待处理完成
        max_wait = 30  # 最多等待30秒
        wait_time = 0
        while wait_time < max_wait:
            if completion_result:
                break
            time.sleep(0.5)
            wait_time += 0.5
        
        if not completion_result:
            print("❌ 处理超时")
            return False
        
        if completion_result.get('success'):
            print(f"✅ 线程处理成功: {completion_result.get('message')}")
            
            # 检查输出文件
            expected_output = os.path.join(output_dir, "test_mock.ogg")
            if os.path.exists(expected_output):
                file_size = os.path.getsize(expected_output)
                print(f"✅ 输出文件存在: {expected_output} ({file_size} 字节)")
                print(f"进度更新次数: {len(progress_updates)}")
                return True
            else:
                print(f"❌ 输出文件不存在: {expected_output}")
                files = os.listdir(output_dir)
                print(f"输出目录文件: {files}")
                return False
        else:
            print(f"❌ 线程处理失败: {completion_result.get('message')}")
            return False
            
    except Exception as e:
        print(f"❌ 测试异常: {e}")
        import traceback
        traceback.print_exc()
        return False
    finally:
        # 清理
        if temp_dir and os.path.exists(temp_dir):
            shutil.rmtree(temp_dir)
        if 'output_dir' in locals() and os.path.exists(output_dir):
            shutil.rmtree(output_dir)

def test_batch_processing():
    """测试批处理模式"""
    print("\n" + "=" * 60)
    print("测试3: 批处理模式测试")
    print("=" * 60)
    
    # 创建多个模拟mgg文件
    mock_files = []
    temp_dirs = []
    
    try:
        for i in range(3):
            mock_mgg_file, temp_dir = create_mock_mgg_file()
            if mock_mgg_file:
                # 重命名为不同的文件
                new_name = os.path.join(temp_dir, f"test_mock_{i}.mgg")
                os.rename(mock_mgg_file, new_name)
                mock_files.append(new_name)
                temp_dirs.append(temp_dir)
        
        if not mock_files:
            print("❌ 无法创建模拟文件")
            return False
        
        # 创建输出目录
        output_dir = tempfile.mkdtemp(prefix="mgg_batch_output_")
        
        # 创建处理器
        processor = FileProcessor("./um.exe", use_service_mode=False)
        
        print(f"批处理文件: {[os.path.basename(f) for f in mock_files]}")
        print(f"输出目录: {output_dir}")
        
        # 批处理
        result = processor.process_files_batch(
            mock_files,
            output_dir=output_dir,
            use_source_dir=False,
            naming_format="auto"
        )
        
        print(f"批处理结果: {result}")
        
        if result.get('success_count', 0) > 0:
            print(f"✅ 批处理成功: {result.get('success_count')} 个文件")
            
            # 检查输出文件
            output_files = os.listdir(output_dir)
            print(f"输出文件: {output_files}")
            
            expected_count = len(mock_files)
            actual_count = len([f for f in output_files if f.endswith('.ogg')])
            
            if actual_count >= expected_count:
                print(f"✅ 输出文件数量正确: {actual_count}/{expected_count}")
                return True
            else:
                print(f"❌ 输出文件数量不足: {actual_count}/{expected_count}")
                return False
        else:
            print(f"❌ 批处理失败: {result}")
            return False
            
    except Exception as e:
        print(f"❌ 测试异常: {e}")
        import traceback
        traceback.print_exc()
        return False
    finally:
        # 清理
        for temp_dir in temp_dirs:
            if os.path.exists(temp_dir):
                shutil.rmtree(temp_dir)
        if 'output_dir' in locals() and os.path.exists(output_dir):
            shutil.rmtree(output_dir)

def main():
    """主函数"""
    print("GUI MGG文件处理修复验证脚本")
    print("模拟真实的GUI环境来测试mgg文件处理修复")
    print("=" * 60)
    
    # 检查um.exe
    if not os.path.exists("./um.exe"):
        print("❌ um.exe文件不存在")
        sys.exit(1)
    
    # 检查测试数据
    if not os.path.exists("./tests/data/test.mflac"):
        print("❌ 测试数据文件不存在")
        sys.exit(1)
    
    # 运行测试
    results = []
    
    print("开始测试...")
    
    # 测试1: 直接处理器测试
    result1 = test_processor_direct()
    results.append(("直接FileProcessor", result1))
    
    # 测试2: 线程管理器测试
    result2 = test_thread_manager()
    results.append(("ThreadManager（模拟GUI）", result2))
    
    # 测试3: 批处理测试
    result3 = test_batch_processing()
    results.append(("批处理模式", result3))
    
    # 总结
    print("\n" + "=" * 60)
    print("测试总结")
    print("=" * 60)
    
    all_passed = True
    for test_name, result in results:
        status = "✅ 通过" if result else "❌ 失败"
        print(f"{test_name}: {status}")
        if not result:
            all_passed = False
    
    if all_passed:
        print("\n🎉 所有测试通过！MGG文件处理修复成功！")
        print("GUI现在应该能正确处理mgg文件了。")
        sys.exit(0)
    else:
        print("\n💥 部分测试失败！需要进一步调试。")
        sys.exit(1)

if __name__ == "__main__":
    main()
