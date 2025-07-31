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

# 拖拽功能支持
try:
    from tkinterdnd2 import DND_FILES, DND_ALL, TkinterDnD
    DRAG_DROP_AVAILABLE = True
except ImportError:
    DRAG_DROP_AVAILABLE = False
    TkinterDnD = None

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

        # 拖拽支持已在创建窗口时初始化（如果使用 TkinterDnD.Tk）

        # 创建组件
        self.setup_ui()
        self.setup_drag_drop()
        
        # 初始化处理器和线程管理器
        self.processor = FileProcessor(um_exe_path)
        self.thread_manager = ThreadManager(max_workers=DEFAULT_MAX_WORKERS)

        # 获取支持的格式列表
        self.supported_extensions = self.processor.supported_extensions

        # 消息队列用于线程间通信
        self.message_queue = queue.Queue()
        self.check_queue()

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

        # 配置拖拽样式
        self._setup_drag_styles()
        
        # 主框架
        main_frame = ttk.Frame(self.root, padding="10")
        main_frame.grid(row=0, column=0, sticky=(tk.W, tk.E, tk.N, tk.S))
        
        # 配置网格权重
        self.root.columnconfigure(0, weight=1)
        self.root.rowconfigure(0, weight=1)
        main_frame.columnconfigure(1, weight=1)
        main_frame.rowconfigure(2, weight=1)
        
        # 输出目录选择
        output_frame = ttk.LabelFrame(main_frame, text="输出设置", padding="5")
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
        button_frame = ttk.Frame(main_frame)
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
        list_frame = ttk.LabelFrame(main_frame, text="文件列表 (支持拖拽文件到此区域)", padding="5")
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
        status_frame = ttk.Frame(main_frame)
        status_frame.grid(row=3, column=0, columnspan=3, sticky=(tk.W, tk.E), pady=(10, 0))
        status_frame.columnconfigure(1, weight=1)
        
        ttk.Label(status_frame, text="状态:").grid(row=0, column=0, sticky=tk.W)
        self.status_var = tk.StringVar(value="就绪")
        ttk.Label(status_frame, textvariable=self.status_var).grid(row=0, column=1, sticky=tk.W, padx=(5, 0))
        
        # 进度条
        self.progress_var = tk.DoubleVar()
        self.progress_bar = ttk.Progressbar(status_frame, variable=self.progress_var, maximum=100)
        self.progress_bar.grid(row=1, column=0, columnspan=2, sticky=(tk.W, tk.E), pady=(5, 0))

    def _setup_drag_styles(self):
        """设置拖拽相关的样式"""
        if DRAG_DROP_AVAILABLE:
            style = ttk.Style()
            # 配置拖拽进入时的样式
            style.configure('DragEnter.Treeview',
                          fieldbackground='lightblue',
                          background='lightblue')

    def setup_drag_drop(self):
        """设置拖拽功能"""
        if DRAG_DROP_AVAILABLE:
            # 使用 tkinterdnd2 设置拖拽功能
            # 在整个窗口上设置拖拽
            self.root.drop_target_register(DND_ALL)
            self.root.dnd_bind('<<Drop>>', self.on_drop_files)
            self.root.dnd_bind('<<DragEnter>>', self.on_drag_enter)
            self.root.dnd_bind('<<DragLeave>>', self.on_drag_leave)
        else:
            # 如果拖拽库不可用，显示提示信息
            self.setup_manual_drag_info()

    def setup_manual_drag_info(self):
        """设置手动拖拽提示信息"""
        # 在文件列表框架中添加提示文本
        info_label = ttk.Label(
            self.file_tree.master,
            text="提示：如果拖拽不工作，请使用'添加文件'或'添加文件夹'按钮",
            foreground="gray"
        )
        info_label.grid(row=2, column=0, sticky=tk.W, pady=(5, 0))

    def on_drop_files(self, event):
        """处理拖拽文件事件"""
        try:
            # 获取拖拽的文件路径，处理各种格式
            dropped_data = event.data
            print(f"原始拖拽数据: {dropped_data}")  # 调试信息

            # 清理拖拽数据中的特殊字符
            dropped_file = dropped_data.replace("{", "").replace("}", "").strip()

            # 解析多个文件（支持多种分隔符）
            files = []
            if '\n' in dropped_file:
                files = dropped_file.split('\n')
            elif '\r' in dropped_file:
                files = dropped_file.split('\r')
            elif ' ' in dropped_file and len(dropped_file.split(' ')) > 1:
                # 处理空格分隔的多个文件路径
                potential_files = dropped_file.split(' ')
                for f in potential_files:
                    if os.path.exists(f.strip()):
                        files.append(f.strip())
                if not files:  # 如果按空格分割没有找到有效文件，当作单个文件处理
                    files = [dropped_file]
            else:
                files = [dropped_file]

            # 过滤和处理文件
            valid_files = []
            invalid_files = []

            for file_path in files:
                file_path = file_path.strip().strip('"').strip("'")  # 移除引号和空白
                if not file_path:
                    continue

                if os.path.isfile(file_path):
                    # 单个文件
                    if self.processor.is_supported_file(file_path):
                        valid_files.append(file_path)
                    else:
                        invalid_files.append(file_path)
                elif os.path.isdir(file_path):
                    # 文件夹 - 扫描其中的音乐文件
                    folder_files = self.scan_directory(file_path)
                    valid_files.extend(folder_files)
                else:
                    print(f"路径不存在: {file_path}")  # 调试信息

            # 提供详细的反馈
            if valid_files:
                self.add_files_to_list(valid_files)
                status_msg = f"成功添加 {len(valid_files)} 个文件"
                if invalid_files:
                    status_msg += f"，跳过 {len(invalid_files)} 个不支持的文件"
                self.update_status(status_msg)
                return "copy"  # 返回成功的拖拽动作
            else:
                if invalid_files:
                    messagebox.showinfo("提示",
                        f"拖拽的 {len(invalid_files)} 个文件都不是支持的音乐格式\n"
                        f"支持的格式：{', '.join(sorted(self.processor.supported_extensions))}")
                else:
                    messagebox.showinfo("提示", "拖拽的文件中没有找到支持的音乐格式")
                return "copy"  # 即使没有有效文件，也返回成功以避免禁止标志

        except Exception as e:
            error_msg = f"处理拖拽文件时出错：{str(e)}"
            messagebox.showerror("错误", error_msg)
            print(f"拖拽处理错误: {e}")  # 调试信息
            import traceback
            traceback.print_exc()  # 打印完整错误堆栈
            return "copy"  # 即使出错，也返回动作以避免禁止标志

    def on_drag_enter(self, event):
        """处理拖拽进入事件"""
        try:
            # 改变窗口标题提示用户
            original_title = self.root.title()
            if "拖拽文件到此处" not in original_title:
                self.root.title(f"{original_title} - 拖拽文件到此处")
            return "copy"
        except Exception as e:
            print(f"拖拽进入事件错误: {e}")
            return "copy"

    def on_drag_leave(self, event):
        """处理拖拽离开事件"""
        try:
            # 恢复原始窗口标题
            title = self.root.title()
            if " - 拖拽文件到此处" in title:
                self.root.title(title.replace(" - 拖拽文件到此处", ""))
            return "copy"
        except Exception as e:
            print(f"拖拽离开事件错误: {e}")
            return "copy"






    
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
