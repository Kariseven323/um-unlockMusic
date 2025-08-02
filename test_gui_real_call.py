#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试GUI真实调用方式的脚本
模拟GUI实际调用processor.py的方式，找出带参数调用失败的原因
"""

import os
import sys
import tempfile
import logging

# 添加项目路径到sys.path
project_root = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, project_root)

from music_unlock_gui.core.processor import FileProcessor

def setup_detailed_logging():
    """设置详细的日志记录"""
    # 设置根日志级别为DEBUG
    logging.basicConfig(
        level=logging.DEBUG,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        handlers=[
            logging.StreamHandler(sys.stdout)
        ]
    )
    
    # 确保FileProcessor的日志也是DEBUG级别
    logger = logging.getLogger('FileProcessor')
    logger.setLevel(logging.DEBUG)

def test_gui_call_with_real_mgg():
    """测试GUI调用方式处理真实mgg文件"""
    print("=" * 80)
    print("测试GUI调用方式处理真实mgg文件")
    print("=" * 80)
    
    # 真实的mgg文件路径
    mgg_file = r"C:\Users\KAR1SEVEN\Music\VipSongsDownload\魏如萱\藏着并不等于遗忘\既然不再假装自己不是多愁善感的诗人 - 魏如萱.mgg"
    
    if not os.path.exists(mgg_file):
        print(f"❌ mgg文件不存在: {mgg_file}")
        return False
    
    # 创建临时输出目录
    output_dir = tempfile.mkdtemp(prefix="gui_mgg_test_")
    print(f"输出目录: {output_dir}")
    
    try:
        # 创建FileProcessor（不使用服务模式，直接subprocess调用）
        processor = FileProcessor("./um.exe", use_service_mode=False)
        
        print("\n1. 测试源目录模式（GUI默认模式）")
        print("-" * 50)
        
        success1, message1 = processor.process_file(
            mgg_file,
            use_source_dir=True,  # GUI默认使用源目录
            naming_format="auto"
        )
        
        print(f"结果: {'✅ 成功' if success1 else '❌ 失败'}")
        print(f"消息: {message1}")
        
        if success1:
            # 检查源目录是否有输出文件
            source_dir = os.path.dirname(mgg_file)
            expected_output = os.path.join(source_dir, "既然不再假装自己不是多愁善感的诗人 - 魏如萱.ogg")
            if os.path.exists(expected_output):
                file_size = os.path.getsize(expected_output)
                print(f"✅ 源目录输出文件: {expected_output} ({file_size} 字节)")
            else:
                print(f"❌ 源目录未找到输出文件: {expected_output}")
        
        print("\n2. 测试指定输出目录模式")
        print("-" * 50)
        
        success2, message2 = processor.process_file(
            mgg_file,
            output_dir=output_dir,
            use_source_dir=False,
            naming_format="auto"
        )
        
        print(f"结果: {'✅ 成功' if success2 else '❌ 失败'}")
        print(f"消息: {message2}")
        
        if success2:
            # 检查指定目录是否有输出文件
            expected_output = os.path.join(output_dir, "既然不再假装自己不是多愁善感的诗人 - 魏如萱.ogg")
            if os.path.exists(expected_output):
                file_size = os.path.getsize(expected_output)
                print(f"✅ 指定目录输出文件: {expected_output} ({file_size} 字节)")
            else:
                print(f"❌ 指定目录未找到输出文件: {expected_output}")
                # 列出目录中的所有文件
                files = os.listdir(output_dir) if os.path.exists(output_dir) else []
                print(f"目录中的文件: {files}")
        
        print("\n3. 测试不同命名格式")
        print("-" * 50)
        
        naming_formats = ["auto", "original", "title-artist", "artist-title"]
        for naming_format in naming_formats:
            print(f"\n测试命名格式: {naming_format}")
            
            success3, message3 = processor.process_file(
                mgg_file,
                output_dir=output_dir,
                use_source_dir=False,
                naming_format=naming_format
            )
            
            print(f"  结果: {'✅ 成功' if success3 else '❌ 失败'}")
            if not success3:
                print(f"  错误: {message3}")
        
        print("\n4. 测试批处理模式")
        print("-" * 50)
        
        batch_result = processor.process_files_batch(
            [mgg_file],
            output_dir=output_dir,
            use_source_dir=False,
            naming_format="auto"
        )
        
        print(f"批处理结果: {batch_result}")
        
        if batch_result.get('success_count', 0) > 0:
            print("✅ 批处理成功")
        else:
            print("❌ 批处理失败")
            if 'results' in batch_result:
                for result in batch_result['results']:
                    if not result.get('success', False):
                        print(f"  失败文件: {result.get('input_path')}")
                        print(f"  错误: {result.get('error')}")
        
        # 总结
        print("\n" + "=" * 80)
        print("测试总结")
        print("=" * 80)
        
        overall_success = success1 or success2
        if overall_success:
            print("✅ GUI调用方式测试通过")
            print("修复成功！前端GUI现在应该能正确处理mgg文件了。")
        else:
            print("❌ GUI调用方式测试失败")
            print("前端GUI调用仍然存在问题，需要进一步调试。")
        
        return overall_success
        
    except Exception as e:
        print(f"❌ 测试过程中发生异常: {e}")
        import traceback
        traceback.print_exc()
        return False
    
    finally:
        # 清理临时目录
        import shutil
        try:
            shutil.rmtree(output_dir)
            print(f"\n清理临时目录: {output_dir}")
        except Exception as e:
            print(f"清理临时目录失败: {e}")

def test_subprocess_call_directly():
    """直接测试subprocess调用，模拟GUI的确切调用方式"""
    print("\n" + "=" * 80)
    print("直接测试subprocess调用（模拟GUI确切方式）")
    print("=" * 80)
    
    import subprocess
    import platform
    
    mgg_file = r"C:\Users\KAR1SEVEN\Music\VipSongsDownload\魏如萱\藏着并不等于遗忘\既然不再假装自己不是多愁善感的诗人 - 魏如萱.mgg"
    output_dir = tempfile.mkdtemp(prefix="direct_subprocess_test_")
    
    print(f"输入文件: {mgg_file}")
    print(f"输出目录: {output_dir}")
    
    try:
        # 模拟GUI的确切调用方式
        um_exe_path = "./um.exe"
        um_exe_dir = os.path.dirname(os.path.abspath(um_exe_path))
        
        cmd = [
            um_exe_path,
            '-i', mgg_file,
            '-o', output_dir,
            '--naming-format', 'auto',
            '--overwrite',
            '--verbose'
        ]
        
        print(f"执行命令: {' '.join(cmd)}")
        print(f"工作目录: {um_exe_dir}")
        
        # 获取subprocess参数
        kwargs = {}
        if platform.system() == 'Windows':
            kwargs['creationflags'] = subprocess.CREATE_NO_WINDOW
        
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            encoding='utf-8',
            errors='ignore',
            timeout=60,
            cwd=um_exe_dir,
            **kwargs
        )
        
        print(f"\n返回码: {result.returncode}")
        print(f"标准输出:\n{result.stdout}")
        print(f"错误输出:\n{result.stderr}")
        
        if result.returncode == 0:
            print("✅ subprocess调用成功")
            
            # 检查输出文件
            expected_output = os.path.join(output_dir, "既然不再假装自己不是多愁善感的诗人 - 魏如萱.ogg")
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
            print("❌ subprocess调用失败")
            return False
            
    except Exception as e:
        print(f"❌ subprocess调用异常: {e}")
        import traceback
        traceback.print_exc()
        return False
    
    finally:
        # 清理
        import shutil
        try:
            shutil.rmtree(output_dir)
            print(f"\n清理临时目录: {output_dir}")
        except Exception as e:
            print(f"清理临时目录失败: {e}")

def main():
    """主函数"""
    print("GUI真实调用方式测试脚本")
    print("专门测试前端GUI带参数调用后端的问题")
    
    # 设置详细日志
    setup_detailed_logging()
    
    # 检查文件
    if not os.path.exists("./um.exe"):
        print("❌ um.exe文件不存在")
        sys.exit(1)
    
    mgg_file = r"C:\Users\KAR1SEVEN\Music\VipSongsDownload\魏如萱\藏着并不等于遗忘\既然不再假装自己不是多愁善感的诗人 - 魏如萱.mgg"
    if not os.path.exists(mgg_file):
        print(f"❌ mgg文件不存在: {mgg_file}")
        sys.exit(1)
    
    # 运行测试
    print("开始测试前端GUI调用方式...")
    
    # 测试1: GUI调用方式
    result1 = test_gui_call_with_real_mgg()
    
    # 测试2: 直接subprocess调用
    result2 = test_subprocess_call_directly()
    
    # 总结
    print("\n" + "=" * 80)
    print("最终总结")
    print("=" * 80)
    
    print(f"GUI调用方式: {'✅ 成功' if result1 else '❌ 失败'}")
    print(f"直接subprocess调用: {'✅ 成功' if result2 else '❌ 失败'}")
    
    if result1 and result2:
        print("\n🎉 前端GUI调用修复成功！")
        print("现在GUI应该能正确处理mgg文件了。")
        sys.exit(0)
    else:
        print("\n💥 前端GUI调用仍有问题！")
        if not result1:
            print("- GUI调用方式失败")
        if not result2:
            print("- 直接subprocess调用失败")
        sys.exit(1)

if __name__ == "__main__":
    main()
