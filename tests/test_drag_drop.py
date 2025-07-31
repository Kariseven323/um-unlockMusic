#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试拖拽功能的简单脚本
"""

import sys
import os

# 添加项目路径
current_dir = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(current_dir, 'music_unlock_gui'))

def test_tkinterdnd2_import():
    """测试tkinterdnd2导入"""
    try:
        from tkinterdnd2 import TkinterDnD, DND_ALL
        print("✅ tkinterdnd2导入成功")
        return True
    except ImportError as e:
        print(f"❌ tkinterdnd2导入失败: {e}")
        return False

def test_drag_drop_window():
    """测试拖拽窗口创建"""
    try:
        import tkinter as tk
        from tkinterdnd2 import TkinterDnD, DND_ALL
        
        # 创建支持拖拽的窗口类
        class DnDTk(tk.Tk, TkinterDnD.DnDWrapper):
            def __init__(self, *args, **kwargs):
                super().__init__(*args, **kwargs)
                self.TkdndVersion = TkinterDnD._require(self)
        
        root = DnDTk()
        root.title("拖拽测试")
        root.geometry("400x300")
        
        # 设置拖拽
        root.drop_target_register(DND_ALL)
        
        def on_drop(event):
            print(f"拖拽文件: {event.data}")
            return "copy"
        
        root.dnd_bind('<<Drop>>', on_drop)
        
        # 添加说明标签
        label = tk.Label(root, text="请拖拽文件到此窗口测试\n如果能看到文件路径输出，说明拖拽功能正常", 
                        justify=tk.CENTER, pady=50)
        label.pack(expand=True)
        
        print("✅ 拖拽窗口创建成功")
        print("请拖拽文件到窗口测试功能...")
        
        root.mainloop()
        return True
        
    except Exception as e:
        print(f"❌ 拖拽窗口创建失败: {e}")
        return False

if __name__ == "__main__":
    print("开始测试拖拽功能...")
    print("=" * 40)
    
    # 测试导入
    if test_tkinterdnd2_import():
        # 测试窗口
        test_drag_drop_window()
    else:
        print("无法继续测试，请检查tkinterdnd2安装")
