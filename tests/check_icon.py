#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
检查exe文件是否包含图标资源
"""

import os
import sys

def check_exe_icon(exe_path):
    """检查exe文件是否包含图标"""
    if not os.path.exists(exe_path):
        print(f"文件不存在: {exe_path}")
        return False
    
    try:
        # 在Windows上，可以使用win32api检查图标
        import win32api
        import win32con
        
        # 尝试提取图标
        icon_count = win32api.ExtractIcon(0, exe_path, -1)
        print(f"文件 {exe_path} 包含 {icon_count} 个图标")
        
        if icon_count > 0:
            print("✓ 图标已正确嵌入到exe文件中")
            return True
        else:
            print("✗ exe文件中未找到图标")
            return False
            
    except ImportError:
        print("需要安装pywin32: pip install pywin32")
        return False
    except Exception as e:
        print(f"检查图标时出错: {e}")
        return False

def main():
    """主函数"""
    exe_path = "dist/MusicUnlockGUI.exe"
    
    if len(sys.argv) > 1:
        exe_path = sys.argv[1]
    
    print(f"检查exe文件图标: {exe_path}")
    print("=" * 50)
    
    check_exe_icon(exe_path)

if __name__ == "__main__":
    main()
