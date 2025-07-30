# 测试文件目录

本目录包含了项目的所有测试相关文件，按功能分类整理。

## 📁 目录结构

```
tests/
├── scripts/          # 测试脚本
├── data/            # 测试数据文件
├── configs/         # 测试配置文件
├── reports/         # 测试报告文件
└── README.md        # 本说明文件
```

## 📝 文件说明

### scripts/ - 测试脚本
包含各种功能测试脚本：

**Python 测试脚本**：
- `test_batch_mode.py` - 批处理模式测试
- `test_complete_functionality.py` - 完整功能测试
- `test_formats.py` - 格式支持测试
- `test_gui_batch_integration.py` - GUI批处理集成测试
- `test_naming_functionality.py` - 文件命名功能测试
- `test_performance_optimization.py` - 性能优化测试
- `test_*_optimizations.py` - 各级别优化测试

**Go 测试脚本**：
- `test_basic_optimizations.go` - 基础优化测试
- `test_optimizations_simple.go` - 简单优化测试

### data/ - 测试数据文件
- `test.mflac` - 测试用的加密音频文件
- `test.flac` - 测试用的解密音频文件

### configs/ - 测试配置文件
- `test_batch.json` - 批处理测试配置
- `test_simple.json` - 简单测试配置
- `test_real_file.json` - 真实文件测试配置
- `test_batch_request.json` - 批处理请求配置

### reports/ - 测试报告文件
- `performance_test_results.json` - 性能测试结果
- `optimization_verification_report.md` - 优化验证报告
- `*_optimization_report.*` - 各级别优化报告

## 🚀 运行测试

### 运行单个测试
```bash
# Python 测试
python tests/scripts/test_batch_mode.py

# Go 测试
go run tests/scripts/test_basic_optimizations.go
```

### 批量运行测试
```bash
# 运行所有 Python 测试
for test in tests/scripts/test_*.py; do
    echo "运行: $test"
    python "$test"
done
```

## 📋 测试类型

1. **功能测试** - 验证核心功能是否正常工作
2. **性能测试** - 测试处理速度和资源使用
3. **格式测试** - 验证各种音频格式支持
4. **集成测试** - 测试GUI和CLI的集成
5. **优化测试** - 验证性能优化效果

## 📌 注意事项

- 运行测试前请确保 `um.exe` 在项目根目录
- 某些测试需要 `testdata/` 目录中的测试文件
- 测试结果会保存在 `reports/` 目录中
- 建议在测试前备份重要数据

## 🔗 相关文档

- [项目主文档](../README.md)
- [性能优化总结](../OPTIMIZATION_SUMMARY.md)
- [综合优化总结](../comprehensive_optimization_summary.md)
