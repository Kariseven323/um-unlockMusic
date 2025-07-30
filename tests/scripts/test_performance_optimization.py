#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
性能优化测试和验证
测试批处理模式、内存池优化和并发处理的性能提升效果
"""

import json
import subprocess
import os
import time
import tempfile
import shutil
from typing import List, Dict, Any
import statistics

class PerformanceTest:
    """性能测试类"""
    
    def __init__(self, um_exe_path: str = "um_new.exe"):
        self.um_exe_path = um_exe_path
        self.test_results = {}
        
    def create_test_files(self, count: int = 10) -> List[str]:
        """
        创建测试文件（复制现有的测试文件）
        
        Args:
            count: 要创建的测试文件数量
            
        Returns:
            List[str]: 测试文件路径列表
        """
        test_files = []
        base_file = "testdata/听妈妈的话 (Live) - 周杰伦 _ 潘玮柏 _ 张学友.mflac"
        
        if not os.path.exists(base_file):
            print(f"警告: 基础测试文件 {base_file} 不存在，创建模拟文件")
            # 创建模拟文件用于测试
            os.makedirs("testdata", exist_ok=True)
            with open(base_file, 'wb') as f:
                f.write(b'MOCK_MFLAC_FILE' * 1000)  # 创建一个小的模拟文件
        
        # 创建临时目录
        temp_dir = tempfile.mkdtemp(prefix="um_perf_test_")
        
        try:
            for i in range(count):
                test_file = os.path.join(temp_dir, f"test_file_{i:03d}.mflac")
                shutil.copy2(base_file, test_file)
                test_files.append(test_file)
        except Exception as e:
            print(f"创建测试文件失败: {e}")
            # 清理已创建的文件
            if os.path.exists(temp_dir):
                shutil.rmtree(temp_dir)
            return []
        
        return test_files
    
    def cleanup_test_files(self, test_files: List[str]):
        """清理测试文件"""
        if test_files:
            temp_dir = os.path.dirname(test_files[0])
            if os.path.exists(temp_dir):
                shutil.rmtree(temp_dir)
    
    def test_traditional_mode(self, test_files: List[str]) -> Dict[str, Any]:
        """
        测试传统模式（逐个文件处理）
        
        Args:
            test_files: 测试文件列表
            
        Returns:
            Dict[str, Any]: 测试结果
        """
        print(f"\n=== 传统模式测试 ({len(test_files)} 个文件) ===")
        
        output_dir = tempfile.mkdtemp(prefix="um_traditional_")
        start_time = time.time()
        
        success_count = 0
        failed_count = 0
        process_times = []
        
        try:
            for i, test_file in enumerate(test_files):
                file_start = time.time()
                
                cmd = [
                    self.um_exe_path,
                    '-i', test_file,
                    '-o', output_dir,
                    '--overwrite'
                ]
                
                try:
                    result = subprocess.run(
                        cmd,
                        capture_output=True,
                        text=True,
                        encoding='utf-8',
                        timeout=30
                    )
                    
                    file_time = time.time() - file_start
                    process_times.append(file_time)
                    
                    if result.returncode == 0:
                        success_count += 1
                        print(f"  文件 {i+1}: 成功 ({file_time:.2f}s)")
                    else:
                        failed_count += 1
                        print(f"  文件 {i+1}: 失败 - {result.stderr}")
                        
                except subprocess.TimeoutExpired:
                    failed_count += 1
                    print(f"  文件 {i+1}: 超时")
                    process_times.append(30.0)
                except Exception as e:
                    failed_count += 1
                    print(f"  文件 {i+1}: 异常 - {e}")
                    process_times.append(0.0)
            
            total_time = time.time() - start_time
            
            result = {
                'mode': 'traditional',
                'total_files': len(test_files),
                'success_count': success_count,
                'failed_count': failed_count,
                'total_time': total_time,
                'avg_time_per_file': statistics.mean(process_times) if process_times else 0,
                'min_time': min(process_times) if process_times else 0,
                'max_time': max(process_times) if process_times else 0,
                'process_times': process_times
            }
            
            print(f"传统模式结果:")
            print(f"  总耗时: {total_time:.2f}s")
            print(f"  成功: {success_count}, 失败: {failed_count}")
            print(f"  平均每文件: {result['avg_time_per_file']:.2f}s")
            
            return result
            
        finally:
            # 清理输出目录
            if os.path.exists(output_dir):
                shutil.rmtree(output_dir)
    
    def test_batch_mode(self, test_files: List[str]) -> Dict[str, Any]:
        """
        测试批处理模式
        
        Args:
            test_files: 测试文件列表
            
        Returns:
            Dict[str, Any]: 测试结果
        """
        print(f"\n=== 批处理模式测试 ({len(test_files)} 个文件) ===")
        
        output_dir = tempfile.mkdtemp(prefix="um_batch_")
        start_time = time.time()
        
        try:
            # 构建批处理请求
            batch_request = {
                "files": [
                    {"input_path": file_path, "output_path": output_dir}
                    for file_path in test_files
                ],
                "options": {
                    "remove_source": False,
                    "update_metadata": True,
                    "overwrite_output": True,
                    "skip_noop": True
                }
            }
            
            # 执行批处理
            cmd = [self.um_exe_path, "--batch"]
            
            result = subprocess.run(
                cmd,
                input=json.dumps(batch_request),
                capture_output=True,
                text=True,
                encoding='utf-8',
                timeout=len(test_files) * 30  # 根据文件数量调整超时
            )
            
            total_time = time.time() - start_time
            
            if result.returncode == 0:
                # 解析批处理响应 - 提取JSON部分（跳过日志行）
                stdout_text = result.stdout.strip()

                # 查找JSON开始位置（第一个独立的{）
                json_start = -1
                for i, char in enumerate(stdout_text):
                    if char == '{' and (i == 0 or stdout_text[i-1] in '\n\r'):
                        json_start = i
                        break

                if json_start >= 0:
                    json_text = stdout_text[json_start:]
                    response = json.loads(json_text)
                else:
                    print(f"调试信息 - 完整输出:")
                    print(f"stdout: {repr(result.stdout)}")
                    print(f"stderr: {repr(result.stderr)}")
                    raise ValueError("未找到有效的JSON响应")
                
                batch_result = {
                    'mode': 'batch',
                    'total_files': response.get('total_files', len(test_files)),
                    'success_count': response.get('success_count', 0),
                    'failed_count': response.get('failed_count', 0),
                    'total_time': total_time,
                    'batch_time_ms': response.get('total_time_ms', 0),
                    'avg_time_per_file': total_time / len(test_files) if test_files else 0,
                    'results': response.get('results', [])
                }
                
                print(f"批处理模式结果:")
                print(f"  总耗时: {total_time:.2f}s")
                print(f"  Go端耗时: {batch_result['batch_time_ms']}ms")
                print(f"  成功: {batch_result['success_count']}, 失败: {batch_result['failed_count']}")
                print(f"  平均每文件: {batch_result['avg_time_per_file']:.2f}s")
                
                return batch_result
            else:
                print(f"批处理失败: {result.stderr}")
                return {
                    'mode': 'batch',
                    'total_files': len(test_files),
                    'success_count': 0,
                    'failed_count': len(test_files),
                    'total_time': total_time,
                    'error': result.stderr
                }
                
        except subprocess.TimeoutExpired:
            print("批处理超时")
            return {
                'mode': 'batch',
                'total_files': len(test_files),
                'success_count': 0,
                'failed_count': len(test_files),
                'total_time': time.time() - start_time,
                'error': '超时'
            }
        except Exception as e:
            print(f"批处理异常: {e}")
            return {
                'mode': 'batch',
                'total_files': len(test_files),
                'success_count': 0,
                'failed_count': len(test_files),
                'total_time': time.time() - start_time,
                'error': str(e)
            }
        finally:
            # 清理输出目录
            if os.path.exists(output_dir):
                shutil.rmtree(output_dir)
    
    def compare_performance(self, traditional_result: Dict[str, Any], 
                          batch_result: Dict[str, Any]) -> Dict[str, Any]:
        """
        比较性能结果
        
        Args:
            traditional_result: 传统模式结果
            batch_result: 批处理模式结果
            
        Returns:
            Dict[str, Any]: 性能比较结果
        """
        print(f"\n=== 性能比较 ===")
        
        if traditional_result['total_time'] > 0:
            time_improvement = (traditional_result['total_time'] - batch_result['total_time']) / traditional_result['total_time'] * 100
        else:
            time_improvement = 0
        
        comparison = {
            'traditional_time': traditional_result['total_time'],
            'batch_time': batch_result['total_time'],
            'time_improvement_percent': time_improvement,
            'traditional_success_rate': traditional_result['success_count'] / traditional_result['total_files'] * 100,
            'batch_success_rate': batch_result['success_count'] / batch_result['total_files'] * 100,
            'meets_target': time_improvement >= 60  # 目标是60%性能提升
        }
        
        print(f"传统模式耗时: {comparison['traditional_time']:.2f}s")
        print(f"批处理模式耗时: {comparison['batch_time']:.2f}s")
        print(f"性能提升: {comparison['time_improvement_percent']:.1f}%")
        print(f"传统模式成功率: {comparison['traditional_success_rate']:.1f}%")
        print(f"批处理模式成功率: {comparison['batch_success_rate']:.1f}%")
        print(f"达到目标(60%提升): {'✓' if comparison['meets_target'] else '✗'}")
        
        return comparison
    
    def run_performance_test(self, file_counts: List[int] = [5, 10, 20]) -> Dict[str, Any]:
        """
        运行完整的性能测试
        
        Args:
            file_counts: 要测试的文件数量列表
            
        Returns:
            Dict[str, Any]: 完整测试结果
        """
        print("开始性能优化测试...")
        
        all_results = {}
        
        for count in file_counts:
            print(f"\n{'='*50}")
            print(f"测试 {count} 个文件的处理性能")
            print(f"{'='*50}")
            
            # 创建测试文件
            test_files = self.create_test_files(count)
            if not test_files:
                print(f"无法创建 {count} 个测试文件，跳过此测试")
                continue
            
            try:
                # 测试传统模式
                traditional_result = self.test_traditional_mode(test_files)
                
                # 测试批处理模式
                batch_result = self.test_batch_mode(test_files)
                
                # 比较性能
                comparison = self.compare_performance(traditional_result, batch_result)
                
                all_results[f'{count}_files'] = {
                    'traditional': traditional_result,
                    'batch': batch_result,
                    'comparison': comparison
                }
                
            finally:
                # 清理测试文件
                self.cleanup_test_files(test_files)
        
        return all_results

def main():
    """主函数"""
    print("性能优化测试和验证")
    print("=" * 50)
    
    # 检查um.exe是否存在
    if not os.path.exists("um.exe"):
        print("错误: um.exe 不存在，请先编译项目")
        return False
    
    # 创建性能测试实例
    perf_test = PerformanceTest()
    
    # 运行性能测试
    results = perf_test.run_performance_test([5, 10])  # 测试5个和10个文件
    
    # 输出总结
    print(f"\n{'='*50}")
    print("性能测试总结")
    print(f"{'='*50}")
    
    overall_improvements = []
    for test_name, test_result in results.items():
        comparison = test_result['comparison']
        improvement = comparison['time_improvement_percent']
        overall_improvements.append(improvement)
        
        print(f"{test_name}: {improvement:.1f}% 性能提升")
    
    if overall_improvements:
        avg_improvement = statistics.mean(overall_improvements)
        print(f"\n平均性能提升: {avg_improvement:.1f}%")
        print(f"目标达成: {'✓' if avg_improvement >= 60 else '✗'} (目标: 60%)")
    
    # 保存详细结果
    with open('performance_test_results.json', 'w', encoding='utf-8') as f:
        json.dump(results, f, indent=2, ensure_ascii=False)
    
    print(f"\n详细结果已保存到: performance_test_results.json")
    
    return True

if __name__ == "__main__":
    main()
