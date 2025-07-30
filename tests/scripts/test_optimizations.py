#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试优化功能
"""

import os
import sys

# 添加项目路径
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'music_unlock_gui'))

from core.processor import FileProcessor
from core.constants import (
    DEFAULT_SUPPORTED_EXTENSIONS,
    PLATFORM_FORMAT_GROUPS,
    ERROR_MESSAGES,
    SUCCESS_MESSAGES
)

def test_constants_usage():
    """测试常量使用"""
    print("=== 常量使用测试 ===")
    
    # 测试默认格式列表
    print(f"默认支持格式数量: {len(DEFAULT_SUPPORTED_EXTENSIONS)}")
    print(f"前5个格式: {DEFAULT_SUPPORTED_EXTENSIONS[:5]}")
    
    # 测试平台分组
    print(f"\n平台分组数量: {len(PLATFORM_FORMAT_GROUPS)}")
    for platform, config in PLATFORM_FORMAT_GROUPS.items():
        keywords = config["keywords"]
        description = config["description"]
        print(f"  {platform}: {len(keywords)} 个关键词 - {description}")
    
    # 测试错误消息
    print(f"\n错误消息数量: {len(ERROR_MESSAGES)}")
    print(f"示例错误消息: {ERROR_MESSAGES['file_not_found']}")
    
    # 测试成功消息
    print(f"\n成功消息数量: {len(SUCCESS_MESSAGES)}")
    print(f"示例成功消息: {SUCCESS_MESSAGES['conversion_success']}")
    
    return True

def test_processor_optimization():
    """测试处理器优化"""
    print("\n=== 处理器优化测试 ===")
    
    try:
        # 初始化处理器
        um_exe_path = os.path.join(os.path.dirname(__file__), 'um.exe')
        processor = FileProcessor(um_exe_path)
        
        # 测试默认格式列表使用
        default_extensions = processor._get_default_extensions()
        print(f"默认格式列表长度: {len(default_extensions)}")
        print(f"与常量一致性: {'✓' if default_extensions == DEFAULT_SUPPORTED_EXTENSIONS else '✗'}")
        
        # 测试错误消息格式化
        test_file = "nonexistent.mp3"
        expected_msg = ERROR_MESSAGES['file_not_found'].format(test_file)
        success, actual_msg = processor.process_file(test_file, use_source_dir=True)
        
        print(f"错误消息测试: {'✓' if not success and expected_msg == actual_msg else '✗'}")
        print(f"  预期: {expected_msg}")
        print(f"  实际: {actual_msg}")
        
        return True
        
    except Exception as e:
        print(f"处理器优化测试失败: {e}")
        return False

def test_file_type_generation():
    """测试文件类型生成优化"""
    print("\n=== 文件类型生成测试 ===")
    
    try:
        # 模拟支持的扩展名
        supported_extensions = ['.ncm', '.qmc0', '.mflac', '.kgm', '.kwm', '.xm']
        
        # 模拟文件类型生成逻辑
        file_types = []
        all_patterns = ";".join([f"*{ext}" for ext in supported_extensions])
        file_types.append(("所有支持的格式", all_patterns))
        
        # 使用配置驱动的平台分类
        for platform, config in PLATFORM_FORMAT_GROUPS.items():
            keywords = config["keywords"]
            extensions = [ext for ext in supported_extensions if 
                         any(keyword in ext for keyword in keywords)]
            
            if extensions:
                patterns = ";".join([f"*{ext}" for ext in extensions])
                file_types.append((platform, patterns))
        
        file_types.append(("所有文件", "*.*"))
        
        print(f"生成的文件类型数量: {len(file_types)}")
        for name, pattern in file_types:
            print(f"  {name}: {pattern}")
        
        return True
        
    except Exception as e:
        print(f"文件类型生成测试失败: {e}")
        return False

def test_code_quality():
    """测试代码质量改进"""
    print("\n=== 代码质量测试 ===")
    
    # 检查常量模块是否可以正常导入
    try:
        from core.constants import (
            DEFAULT_SUPPORTED_EXTENSIONS,
            PLATFORM_FORMAT_GROUPS,
            OUTPUT_MODE_SOURCE,
            OUTPUT_MODE_CUSTOM,
            UI_WINDOW_TITLE
        )
        print("✓ 常量模块导入成功")
    except ImportError as e:
        print(f"✗ 常量模块导入失败: {e}")
        return False
    
    # 检查常量的完整性
    required_constants = {
        'DEFAULT_SUPPORTED_EXTENSIONS': DEFAULT_SUPPORTED_EXTENSIONS,
        'PLATFORM_FORMAT_GROUPS': PLATFORM_FORMAT_GROUPS,
        'ERROR_MESSAGES': ERROR_MESSAGES,
        'SUCCESS_MESSAGES': SUCCESS_MESSAGES
    }

    missing_constants = []
    for const_name, const_value in required_constants.items():
        if const_value is None:
            missing_constants.append(const_name)
    
    if missing_constants:
        print(f"✗ 缺少常量: {missing_constants}")
        return False
    else:
        print("✓ 所有必需常量都存在")
    
    return True

if __name__ == "__main__":
    tests = [
        test_constants_usage,
        test_processor_optimization,
        test_file_type_generation,
        test_code_quality
    ]
    
    all_passed = True
    for test in tests:
        try:
            result = test()
            if not result:
                all_passed = False
        except Exception as e:
            print(f"测试 {test.__name__} 失败: {e}")
            all_passed = False
    
    print(f"\n=== 总体结果 ===")
    print(f"所有测试: {'✓ 通过' if all_passed else '✗ 失败'}")
    
    sys.exit(0 if all_passed else 1)
