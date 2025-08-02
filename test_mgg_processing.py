#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
MGG文件处理测试脚本
用于验证修复后的processor.py是否能正确处理mgg文件
"""

import os
import sys
import tempfile
import shutil
from pathlib import Path

# 添加项目路径到sys.path
project_root = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, project_root)

from music_unlock_gui.core.processor import FileProcessor

def test_mgg_processing(um_exe_path, mgg_file):
    """测试mgg文件处理"""
    print("=" * 60)
    print("MGG文件处理测试")
    print("=" * 60)
    
    # 创建临时输出目录
    output_dir = tempfile.mkdtemp(prefix="mgg_test_")
    print(f"临时输出目录: {output_dir}")
    
    try:
        # 创建文件处理器
        processor = FileProcessor(um_exe_path, use_service_mode=False)
        
        # 验证um.exe
        print("\n1. 验证um.exe...")
        is_valid, version_info = processor.validate_um_exe()
        if is_valid:
            print(f"✅ um.exe验证成功: {version_info}")
        else:
            print(f"❌ um.exe验证失败: {version_info}")
            return False
        
        # 检查支持的格式
        print("\n2. 检查支持的格式...")
        success, extensions = processor.get_supported_extensions()
        if success:
            print(f"✅ 获取到 {len(extensions)} 个支持的格式")
            if '.mgg' in extensions:
                print("✅ 支持mgg格式")
            else:
                print("❌ 不支持mgg格式")
                print(f"支持的格式: {sorted(extensions)}")
                return False
        else:
            print(f"❌ 获取支持格式失败")
            return False
        
        # 检查文件是否支持
        print("\n3. 检查文件格式支持...")
        is_supported = processor.is_supported_file(mgg_file)
        if is_supported:
            print(f"✅ 文件格式支持: {mgg_file}")
        else:
            print(f"❌ 文件格式不支持: {mgg_file}")
            # 调试格式支持情况
            debug_info = processor.debug_format_support(mgg_file)
            print(f"调试信息: {debug_info}")
            return False
        
        # 测试1: 使用源目录模式
        print("\n4. 测试源目录模式...")
        success, message = processor.process_file(
            mgg_file,
            use_source_dir=True,
            naming_format="auto"
        )
        
        if success:
            print(f"✅ 源目录模式成功: {message}")
        else:
            print(f"❌ 源目录模式失败: {message}")
        
        # 测试2: 使用指定输出目录模式
        print("\n5. 测试指定输出目录模式...")
        success2, message2 = processor.process_file(
            mgg_file,
            output_dir=output_dir,
            use_source_dir=False,
            naming_format="auto"
        )
        
        if success2:
            print(f"✅ 指定输出目录模式成功: {message2}")
            
            # 检查输出文件
            mgg_basename = os.path.splitext(os.path.basename(mgg_file))[0]
            expected_output = os.path.join(output_dir, f"{mgg_basename}.ogg")
            if os.path.exists(expected_output):
                file_size = os.path.getsize(expected_output)
                print(f"✅ 输出文件存在: {expected_output} ({file_size} 字节)")
            else:
                print(f"❌ 输出文件不存在: {expected_output}")
                # 列出输出目录的文件
                files = os.listdir(output_dir) if os.path.exists(output_dir) else []
                print(f"输出目录文件: {files}")
        else:
            print(f"❌ 指定输出目录模式失败: {message2}")
        
        # 测试3: 不同命名格式
        print("\n6. 测试不同命名格式...")
        naming_formats = ["auto", "original", "title-artist", "artist-title"]
        
        for naming_format in naming_formats:
            print(f"\n测试命名格式: {naming_format}")
            success3, message3 = processor.process_file(
                mgg_file,
                output_dir=output_dir,
                use_source_dir=False,
                naming_format=naming_format
            )
            
            if success3:
                print(f"✅ {naming_format} 格式成功")
            else:
                print(f"❌ {naming_format} 格式失败: {message3}")
        
        # 总结
        print("\n" + "=" * 60)
        print("测试总结")
        print("=" * 60)
        
        overall_success = success or success2
        if overall_success:
            print("✅ MGG文件处理测试通过")
            print("修复成功！GUI现在应该能正确处理mgg文件了。")
        else:
            print("❌ MGG文件处理测试失败")
            print("需要进一步调试和修复。")
        
        return overall_success
        
    except Exception as e:
        print(f"❌ 测试过程中发生异常: {e}")
        import traceback
        traceback.print_exc()
        return False
    
    finally:
        # 清理临时目录
        try:
            shutil.rmtree(output_dir)
            print(f"\n清理临时目录: {output_dir}")
        except Exception as e:
            print(f"清理临时目录失败: {e}")

def main():
    """主函数"""
    print("MGG文件处理测试脚本")
    print("用于验证修复后的processor.py是否能正确处理mgg文件")
    
    # 检查参数
    if len(sys.argv) < 3:
        print("\n用法: python test_mgg_processing.py <um.exe路径> <mgg文件路径>")
        print("示例: python test_mgg_processing.py ./um.exe ./test.mgg")
        sys.exit(1)
    
    um_exe_path = sys.argv[1]
    mgg_file = sys.argv[2]
    
    # 检查文件是否存在
    if not os.path.exists(um_exe_path):
        print(f"❌ um.exe文件不存在: {um_exe_path}")
        sys.exit(1)
    
    if not os.path.exists(mgg_file):
        print(f"❌ mgg文件不存在: {mgg_file}")
        sys.exit(1)
    
    # 运行测试
    success = test_mgg_processing(um_exe_path, mgg_file)
    
    if success:
        print("\n🎉 测试成功！修复生效。")
        sys.exit(0)
    else:
        print("\n💥 测试失败！需要进一步调试。")
        sys.exit(1)

if __name__ == "__main__":
    main()
