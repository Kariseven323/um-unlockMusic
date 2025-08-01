#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
音乐解密GUI - 主界面
"""

import tkinter as tk
from tkinter import ttk, filedialog, messagebox
import os
import threading
import queue
from typing import List, Optional



from core.processor import FileProcessor
from core.thread_manager import ThreadManager
from core.constants import (
    PLATFORM_FORMAT_GROUPS,
    OUTPUT_MODE_SOURCE,
    OUTPUT_MODE_CUSTOM,
    NAMING_FORMAT_AUTO,
    NAMING_FORMAT_TITLE_ARTIST,
    NAMING_FORMAT_ARTIST_TITLE,
    NAMING_FORMAT_ORIGINAL,
    NAMING_FORMAT_LABELS,
    UI_WINDOW_TITLE,
    UI_WINDOW_SIZE,
    UI_WINDOW_MIN_SIZE,
    DEFAULT_MAX_WORKERS,
    ERROR_MESSAGES,
    SUCCESS_MESSAGES
)


class MusicUnlockGUI:
    """音乐解密GUI主界面类"""
    
    def __init__(self, root: tk.Tk, um_exe_path: str):
        self.root = root
        self.um_exe_path = um_exe_path
        self.output_dir = ""
        self.file_list = []
        self.processing = False



        # 初始化处理器和线程管理器（启用服务模式，获得更好的性能）
        self.processor = FileProcessor(um_exe_path, use_service_mode=True)
        self.thread_manager = ThreadManager(max_workers=DEFAULT_MAX_WORKERS)

        # 获取支持的格式列表
        self.supported_extensions = self.processor.supported_extensions

        # 消息队列用于线程间通信
        self.message_queue = queue.Queue()

        # 创建组件（在初始化处理器之后）
        self.setup_ui()

        # 开始检查队列
        self.check_queue()

    def create_menu_bar(self):
        """创建菜单栏"""
        menubar = tk.Menu(self.root)
        self.root.config(menu=menubar)

        # 帮助菜单
        help_menu = tk.Menu(menubar, tearoff=0)
        menubar.add_cascade(label="帮助", menu=help_menu)
        help_menu.add_command(label="关于", command=self.show_about)



    def show_about(self):
        """显示关于对话框"""
        about_text = """音乐解密工具 - Unlock Music GUI

版本：1.0.0
作者：Kariseven323

这是一个用于解密各种加密音乐文件的工具，
支持网易云音乐、QQ音乐、酷狗音乐等多种格式。

GitHub: https://github.com/Kariseven323/um-unlockMusic"""

        messagebox.showinfo("关于", about_text)

    def _generate_file_types(self) -> List[tuple]:
        """
        根据支持的格式生成文件类型过滤器

        Returns:
            List[tuple]: 文件类型过滤器列表
        """
        # 生成所有支持格式的通配符字符串
        all_patterns = ";".join([f"*{ext}" for ext in self.supported_extensions])

        file_types = [
            ("所有支持的格式", all_patterns),
        ]

        # 使用配置驱动的平台分类
        for platform, config in PLATFORM_FORMAT_GROUPS.items():
            keywords = config["keywords"]
            extensions = [ext for ext in self.supported_extensions if
                         any(keyword in ext for keyword in keywords)]

            if extensions:
                patterns = ";".join([f"*{ext}" for ext in extensions])
                file_types.append((platform, patterns))

        file_types.append(("所有文件", "*.*"))
        return file_types
    
    def setup_ui(self):
        """设置用户界面"""
        self.root.title(UI_WINDOW_TITLE)
        self.root.geometry(UI_WINDOW_SIZE)
        self.root.minsize(*UI_WINDOW_MIN_SIZE)

        # 创建菜单栏
        self.create_menu_bar()

        # 创建主框架
        main_frame = ttk.Frame(self.root, padding="10")
        main_frame.grid(row=0, column=0, sticky=(tk.W, tk.E, tk.N, tk.S))

        # 配置网格权重
        self.root.columnconfigure(0, weight=1)
        self.root.rowconfigure(0, weight=1)
        main_frame.columnconfigure(0, weight=1)
        main_frame.rowconfigure(0, weight=1)

        # 创建标签页控件
        self.notebook = ttk.Notebook(main_frame)
        self.notebook.grid(row=0, column=0, sticky=(tk.W, tk.E, tk.N, tk.S))

        # 创建解密功能标签页
        self.create_decrypt_tab()

        # 创建文件清理标签页
        self.create_cleanup_tab()
        
    def create_decrypt_tab(self):
        """创建解密功能标签页"""
        # 创建解密标签页框架
        decrypt_frame = ttk.Frame(self.notebook, padding="10")
        self.notebook.add(decrypt_frame, text="🔓 音乐解密")

        # 配置网格权重
        decrypt_frame.columnconfigure(1, weight=1)
        decrypt_frame.rowconfigure(2, weight=1)
        
        # 输出目录选择
        output_frame = ttk.LabelFrame(decrypt_frame, text="输出设置", padding="5")
        output_frame.grid(row=0, column=0, columnspan=3, sticky=(tk.W, tk.E), pady=(0, 10))
        output_frame.columnconfigure(1, weight=1)

        # 输出模式选择
        self.output_mode_var = tk.StringVar(value=OUTPUT_MODE_SOURCE)
        mode_frame = ttk.Frame(output_frame)
        mode_frame.grid(row=0, column=0, columnspan=3, sticky=(tk.W, tk.E), pady=(0, 5))

        ttk.Label(mode_frame, text="输出模式:").pack(side=tk.LEFT, padx=(0, 10))
        ttk.Radiobutton(mode_frame, text="默认源目录", variable=self.output_mode_var,
                       value=OUTPUT_MODE_SOURCE, command=self.on_output_mode_change).pack(side=tk.LEFT, padx=(0, 10))
        ttk.Radiobutton(mode_frame, text="统一输出目录", variable=self.output_mode_var,
                       value=OUTPUT_MODE_CUSTOM, command=self.on_output_mode_change).pack(side=tk.LEFT)

        # 自定义输出目录
        custom_frame = ttk.Frame(output_frame)
        custom_frame.grid(row=1, column=0, columnspan=3, sticky=(tk.W, tk.E))
        custom_frame.columnconfigure(1, weight=1)

        self.output_label = ttk.Label(custom_frame, text="输出目录:")
        self.output_label.grid(row=0, column=0, sticky=tk.W, padx=(0, 5))

        self.output_var = tk.StringVar()
        self.output_entry = ttk.Entry(custom_frame, textvariable=self.output_var, state="readonly")
        self.output_entry.grid(row=0, column=1, sticky=(tk.W, tk.E), padx=(0, 5))

        self.output_button = ttk.Button(custom_frame, text="选择目录", command=self.select_output_dir)
        self.output_button.grid(row=0, column=2)

        # 命名格式选择
        naming_frame = ttk.Frame(output_frame)
        naming_frame.grid(row=2, column=0, columnspan=3, sticky=(tk.W, tk.E), pady=(5, 0))

        ttk.Label(naming_frame, text="文件命名:").pack(side=tk.LEFT, padx=(0, 10))

        self.naming_format_var = tk.StringVar(value=NAMING_FORMAT_AUTO)
        naming_combo = ttk.Combobox(naming_frame, textvariable=self.naming_format_var,
                                   values=list(NAMING_FORMAT_LABELS.values()),
                                   state="readonly", width=20)
        naming_combo.pack(side=tk.LEFT)

        # 设置组合框的值映射
        self.naming_format_mapping = {v: k for k, v in NAMING_FORMAT_LABELS.items()}
        self.reverse_naming_format_mapping = {k: v for k, v in NAMING_FORMAT_LABELS.items()}

        # 设置默认显示值
        naming_combo.set(NAMING_FORMAT_LABELS[NAMING_FORMAT_AUTO])

        # 初始状态设置
        self.on_output_mode_change()

        # 文件操作按钮
        button_frame = ttk.Frame(decrypt_frame)
        button_frame.grid(row=1, column=0, columnspan=3, sticky=(tk.W, tk.E), pady=(0, 10))
        
        ttk.Button(button_frame, text="添加文件", command=self.add_files).pack(side=tk.LEFT, padx=(0, 5))
        ttk.Button(button_frame, text="添加文件夹", command=self.add_folder).pack(side=tk.LEFT, padx=(0, 5))
        ttk.Button(button_frame, text="清除列表", command=self.clear_list).pack(side=tk.LEFT, padx=(0, 5))
        
        # 分隔符
        ttk.Separator(button_frame, orient='vertical').pack(side=tk.LEFT, fill=tk.Y, padx=10)
        
        self.start_button = ttk.Button(button_frame, text="开始转换", command=self.start_conversion)
        self.start_button.pack(side=tk.LEFT, padx=(0, 5))
        
        self.stop_button = ttk.Button(button_frame, text="停止转换", command=self.stop_conversion, state="disabled")
        self.stop_button.pack(side=tk.LEFT)
        
        # 文件列表
        list_frame = ttk.LabelFrame(decrypt_frame, text="文件列表", padding="5")
        list_frame.grid(row=2, column=0, columnspan=3, sticky=(tk.W, tk.E, tk.N, tk.S))
        list_frame.columnconfigure(0, weight=1)
        list_frame.rowconfigure(0, weight=1)
        
        # 创建Treeview
        columns = ("文件名", "状态", "进度")
        self.file_tree = ttk.Treeview(list_frame, columns=columns, show="tree headings", height=15)
        
        # 设置列
        self.file_tree.heading("#0", text="路径")
        self.file_tree.heading("文件名", text="文件名")
        self.file_tree.heading("状态", text="状态")
        self.file_tree.heading("进度", text="进度")
        
        self.file_tree.column("#0", width=300, minwidth=200)
        self.file_tree.column("文件名", width=200, minwidth=150)
        self.file_tree.column("状态", width=100, minwidth=80)
        self.file_tree.column("进度", width=100, minwidth=80)
        
        # 滚动条
        scrollbar_y = ttk.Scrollbar(list_frame, orient="vertical", command=self.file_tree.yview)
        scrollbar_x = ttk.Scrollbar(list_frame, orient="horizontal", command=self.file_tree.xview)
        self.file_tree.configure(yscrollcommand=scrollbar_y.set, xscrollcommand=scrollbar_x.set)
        
        self.file_tree.grid(row=0, column=0, sticky=(tk.W, tk.E, tk.N, tk.S))
        scrollbar_y.grid(row=0, column=1, sticky=(tk.N, tk.S))
        scrollbar_x.grid(row=1, column=0, sticky=(tk.W, tk.E))
        
        # 状态栏
        status_frame = ttk.Frame(decrypt_frame)
        status_frame.grid(row=3, column=0, columnspan=3, sticky=(tk.W, tk.E), pady=(10, 0))
        status_frame.columnconfigure(1, weight=1)

        ttk.Label(status_frame, text="状态:").grid(row=0, column=0, sticky=tk.W)
        self.status_var = tk.StringVar(value="就绪")
        ttk.Label(status_frame, textvariable=self.status_var).grid(row=0, column=1, sticky=tk.W, padx=(5, 0))

        # 进度条
        self.progress_var = tk.DoubleVar()
        self.progress_bar = ttk.Progressbar(status_frame, variable=self.progress_var, maximum=100)
        self.progress_bar.grid(row=1, column=0, columnspan=2, sticky=(tk.W, tk.E), pady=(5, 0))

    def create_cleanup_tab(self):
        """创建文件清理标签页"""
        # 创建清理标签页框架
        cleanup_frame = ttk.Frame(self.notebook, padding="10")
        self.notebook.add(cleanup_frame, text="🗑️ 文件清理")

        # 导入并创建删除工具组件
        try:
            from gui.delete_tool_window import DeleteToolEmbedded

            # 创建删除工具的内嵌版本
            self.delete_tool = DeleteToolEmbedded(cleanup_frame, self.supported_extensions)

        except ImportError as e:
            # 如果导入失败，显示错误信息
            error_label = ttk.Label(cleanup_frame, text=f"无法加载文件清理功能：{str(e)}")
            error_label.pack(expand=True)
        except Exception as e:
            # 处理其他可能的错误
            error_label = ttk.Label(cleanup_frame, text=f"文件清理功能初始化失败：{str(e)}")
            error_label.pack(expand=True)
    



    
    def on_output_mode_change(self):
        """输出模式变化处理"""
        mode = self.output_mode_var.get()
        if mode == OUTPUT_MODE_SOURCE:
            # 默认源目录模式 - 禁用自定义目录选择
            self.output_label.config(state="disabled")
            self.output_entry.config(state="disabled")
            self.output_button.config(state="disabled")
            self.output_var.set("(将输出到各文件的源目录)")
        else:
            # 统一输出目录模式 - 启用自定义目录选择
            self.output_label.config(state="normal")
            self.output_entry.config(state="readonly")
            self.output_button.config(state="normal")
            if not self.output_dir:
                self.output_var.set("")
            else:
                self.output_var.set(self.output_dir)

    def select_output_dir(self):
        """选择输出目录"""
        directory = filedialog.askdirectory(title="选择输出目录")
        if directory:
            self.output_dir = directory
            self.output_var.set(directory)

    def get_naming_format(self):
        """获取当前选择的命名格式"""
        display_value = self.naming_format_var.get()
        return self.naming_format_mapping.get(display_value, NAMING_FORMAT_AUTO)
    
    def add_files(self):
        """添加文件"""
        files = filedialog.askopenfilenames(
            title="选择音乐文件",
            filetypes=self._generate_file_types()
        )
        if files:
            self.add_files_to_list(files)
    
    def add_folder(self):
        """添加文件夹"""
        directory = filedialog.askdirectory(title="选择包含音乐文件的文件夹")
        if directory:
            files = self.scan_directory(directory)
            if files:
                self.add_files_to_list(files)
            else:
                messagebox.showinfo("提示", "所选文件夹中没有找到支持的音乐文件")
    
    def scan_directory(self, directory: str) -> List[str]:
        """扫描目录中的音乐文件"""
        files = []

        for root, dirs, filenames in os.walk(directory):
            for filename in filenames:
                if self.processor.is_supported_file(os.path.join(root, filename)):
                    files.append(os.path.join(root, filename))

        return files
    
    def add_files_to_list(self, files: List[str]):
        """添加文件到列表"""
        for file_path in files:
            if file_path not in self.file_list:
                self.file_list.append(file_path)
                filename = os.path.basename(file_path)
                item_id = self.file_tree.insert("", "end", text=file_path, 
                                               values=(filename, "等待", "0%"))
        
        self.update_status(SUCCESS_MESSAGES['files_added'].format(len(files), len(self.file_list)))
    
    def clear_list(self):
        """清除文件列表"""
        if self.processing:
            messagebox.showwarning("警告", ERROR_MESSAGES['already_processing'])
            return

        self.file_list.clear()
        for item in self.file_tree.get_children():
            self.file_tree.delete(item)

        self.update_status(SUCCESS_MESSAGES['list_cleared'])
        self.progress_var.set(0)
    
    def start_conversion(self):
        """开始转换"""
        if not self.file_list:
            messagebox.showwarning("警告", ERROR_MESSAGES['no_files_selected'])
            return

        # 检查输出模式
        output_mode = self.output_mode_var.get()
        if output_mode == OUTPUT_MODE_CUSTOM and not self.output_dir:
            messagebox.showwarning("警告", ERROR_MESSAGES['no_output_dir'])
            return
        
        self.processing = True
        self.start_button.config(state="disabled")
        self.stop_button.config(state="normal")
        
        # 重置所有文件状态
        for item in self.file_tree.get_children():
            self.file_tree.set(item, "状态", "等待")
            self.file_tree.set(item, "进度", "0%")
        
        # 开始处理（使用批处理模式）
        output_mode = self.output_mode_var.get()
        output_dir = self.output_dir if output_mode == OUTPUT_MODE_CUSTOM else None

        # 获取命名格式
        naming_format = self.get_naming_format()

        self.thread_manager.start_batch_processing(
            self.file_list,
            output_dir,
            self.processor,
            self.message_queue,
            use_source_dir=(output_mode == OUTPUT_MODE_SOURCE),
            naming_format=naming_format
        )
        
        self.update_status(SUCCESS_MESSAGES['processing_started'])
    
    def stop_conversion(self):
        """停止转换"""
        self.thread_manager.stop_all()
        self.processing = False
        self.start_button.config(state="normal")
        self.stop_button.config(state="disabled")
        self.update_status(SUCCESS_MESSAGES['processing_stopped'])
    
    def update_status(self, message: str):
        """更新状态信息"""
        self.status_var.set(message)
    
    def is_processing(self) -> bool:
        """检查是否正在处理"""
        return self.processing
    
    def stop_all_tasks(self):
        """停止所有任务"""
        self.thread_manager.stop_all()
    
    def check_queue(self):
        """检查消息队列"""
        try:
            while True:
                message = self.message_queue.get_nowait()
                self.handle_message(message)
        except queue.Empty:
            pass
        
        # 每100ms检查一次队列
        self.root.after(100, self.check_queue)
    
    def handle_message(self, message: dict):
        """处理来自工作线程的消息"""
        msg_type = message.get('type')
        file_path = message.get('file_path')

        # 查找对应的树项
        item_id = None
        if file_path:
            for item in self.file_tree.get_children():
                if self.file_tree.item(item, 'text') == file_path:
                    item_id = item
                    break

        if msg_type == 'progress':
            if item_id:
                self.file_tree.set(item_id, "状态", "处理中")
                self.file_tree.set(item_id, "进度", f"{message.get('progress', 0)}%")

        elif msg_type == 'success':
            if item_id:
                self.file_tree.set(item_id, "状态", "完成")
                self.file_tree.set(item_id, "进度", "100%")

        elif msg_type == 'error':
            if item_id:
                self.file_tree.set(item_id, "状态", "失败")
                self.file_tree.set(item_id, "进度", "错误")

        elif msg_type == 'all_complete':
            self.processing = False
            self.start_button.config(state="normal")
            self.stop_button.config(state="disabled")
            self.update_status("转换完成")
            messagebox.showinfo("完成", "所有文件转换完成！")

        # 批处理模式消息处理
        elif msg_type == 'batch_start':
            total_files = message.get('total_files', 0)
            self.update_status(f"开始批处理 {total_files} 个文件...")

        elif msg_type == 'file_complete':
            if item_id:
                success = message.get('success', False)
                if success:
                    self.file_tree.set(item_id, "状态", "完成")
                    self.file_tree.set(item_id, "进度", "100%")
                else:
                    self.file_tree.set(item_id, "状态", "失败")
                    self.file_tree.set(item_id, "进度", "错误")

        elif msg_type == 'batch_complete':
            self.processing = False
            self.start_button.config(state="normal")
            self.stop_button.config(state="disabled")
            success_count = message.get('success_count', 0)
            failed_count = message.get('failed_count', 0)
            total_time = message.get('total_time', 0)
            self.update_status(f"批处理完成：成功 {success_count} 个，失败 {failed_count} 个")
            messagebox.showinfo("完成", f"批处理完成！\n成功：{success_count} 个\n失败：{failed_count} 个\n耗时：{total_time}ms")

        elif msg_type == 'batch_error':
            self.processing = False
            self.start_button.config(state="normal")
            self.stop_button.config(state="disabled")
            error_msg = message.get('error', '批处理失败')
            self.update_status(f"批处理错误：{error_msg}")
            messagebox.showerror("错误", f"批处理失败：{error_msg}")

        # 更新总体进度
        self.update_overall_progress()
    
    def update_overall_progress(self):
        """更新总体进度"""
        if not self.file_list:
            return
        
        completed = 0
        for item in self.file_tree.get_children():
            status = self.file_tree.set(item, "状态")
            if status in ["完成", "失败"]:
                completed += 1
        
        progress = (completed / len(self.file_list)) * 100
        self.progress_var.set(progress)
