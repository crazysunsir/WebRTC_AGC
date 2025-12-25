# 在 Go 代码中使用 dylib 文件 - 完整说明

## 一、dylib 文件位置

编译后，dylib 文件位于：
```
WebRTC_AGC/build/libagc.dylib
```

## 二、Go 代码中的配置

### 1. CGO 配置（在 `go/agc/agc.go` 中）

```go
package agc

/*
#cgo CFLAGS: -I${SRCDIR}/../..
// 链接到编译好的 dylib 文件
// -L${SRCDIR}/../../build: 指定库文件搜索路径（build 目录）
// -lagc: 链接 libagc.dylib 库
// -lm: 链接数学库
// -Wl,-rpath,${SRCDIR}/../../build: 设置运行时库搜索路径，确保能找到 dylib
#cgo LDFLAGS: -L${SRCDIR}/../../build -lagc -lm -Wl,-rpath,${SRCDIR}/../../build
#include "agc_wrapper.h"
#include <stdlib.h>
*/
import "C"
```

### 2. 关键参数说明

- **`-L${SRCDIR}/../../build`**: 
  - 指定编译时库文件搜索路径
  - `${SRCDIR}` 是当前 Go 源文件所在目录
  - 指向 `build/` 目录，dylib 文件所在位置

- **`-lagc`**: 
  - 链接名为 `libagc.dylib` 的库
  - 链接器会自动添加 `lib` 前缀和 `.dylib` 后缀

- **`-Wl,-rpath,${SRCDIR}/../../build`**: 
  - 设置运行时库搜索路径
  - 确保程序运行时能找到 dylib 文件
  - `-Wl,` 表示将参数传递给链接器

## 三、使用示例

### 基本使用

```go
package main

import (
    "webrtc-agc/agc"
)

func main() {
    // 1. 创建配置
    config := &agc.Config{
        CompressionGaindB: 12,
        TargetLevelDbfs:   1,
        LimiterEnable:     true,
        Mode:              agc.ModeAdaptiveDigital,
    }
    
    // 2. 创建 AGC 实例（内部调用 dylib 中的 AgcCreate() 和 AgcInit()）
    agcInstance, err := agc.NewAGC(16000, config)
    if err != nil {
        panic(err)
    }
    defer agcInstance.Close() // 释放 dylib 中的资源
    
    // 3. 处理音频（内部调用 dylib 中的 AgcProcess()）
    samples := []int16{...} // 你的音频数据
    err = agcInstance.Process(samples)
    if err != nil {
        panic(err)
    }
}
```

## 四、工作流程

### 编译时

1. **CGO 预处理**：
   - 解析 `#cgo` 指令
   - 设置编译和链接参数

2. **编译 C 代码**：
   - 编译 `agc_wrapper.c` 和 `agc.c`
   - 使用 `-I` 参数指定头文件路径

3. **链接 dylib**：
   - 使用 `-L` 参数在 `build/` 目录查找库
   - 使用 `-lagc` 链接 `libagc.dylib`
   - 嵌入 rpath 信息到可执行文件

### 运行时

1. **加载 dylib**：
   - 程序启动时，动态链接器根据 rpath 查找 dylib
   - 加载 `libagc.dylib` 到内存

2. **调用函数**：
   - Go 代码调用 `agc.NewAGC()` 
   - 内部通过 CGO 调用 dylib 中的 `AgcCreate()` 和 `AgcInit()`
   - 所有 AGC 操作都通过 dylib 中的 C 函数完成

3. **释放资源**：
   - 调用 `agcInstance.Close()`
   - 内部调用 dylib 中的 `AgcFree()`

## 五、验证 dylib 链接

### 方法 1: 使用 otool（macOS）

```bash
# 查看可执行文件链接的库
otool -L go/example/example

# 输出示例：
# go/example/example:
#     @rpath/libagc.dylib (compatibility version 1.0.0, current version 1.0.0)
#     /usr/lib/libSystem.B.dylib (compatibility version 1.0.0, current version 1292.100.5)
```

### 方法 2: 使用 nm

```bash
# 查看 dylib 中导出的符号
nm -gU build/libagc.dylib | grep Agc

# 应该看到：
# _AgcCreate
# _AgcFree
# _AgcInit
# _AgcProcess
# _AgcSetConfig
```

### 方法 3: 运行时检查

如果 dylib 未正确链接，运行时会报错：
```
dyld: Library not loaded: @rpath/libagc.dylib
  Referenced from: /path/to/example
  Reason: image not found
```

## 六、路径说明

### 相对路径解析

在 `go/agc/agc.go` 中：
- `${SRCDIR}` = `/Users/sunxuguang/WebRTC_AGC/go/agc`
- `${SRCDIR}/../..` = `/Users/sunxuguang/WebRTC_AGC`
- `${SRCDIR}/../../build` = `/Users/sunxuguang/WebRTC_AGC/build`

### 绝对路径（不推荐）

如果使用绝对路径：
```go
#cgo LDFLAGS: -L/Users/sunxuguang/WebRTC_AGC/build -lagc -lm
```
这样会降低代码的可移植性。

## 七、常见问题

### Q1: 找不到 dylib 文件

**错误信息：**
```
ld: library 'agc' not found
```

**解决方案：**
1. 确保已运行 `./build_so.sh` 构建 dylib
2. 检查 `build/libagc.dylib` 是否存在
3. 验证 `-L` 路径是否正确

### Q2: 运行时找不到 dylib

**错误信息：**
```
dyld: Library not loaded: @rpath/libagc.dylib
```

**解决方案：**
1. 确保 `-Wl,-rpath` 参数已设置
2. 检查 rpath 路径是否正确
3. 使用 `otool -L` 验证 rpath

### Q3: 符号未找到

**错误信息：**
```
undefined symbol: _AgcCreate
```

**解决方案：**
1. 确保 dylib 已正确编译
2. 检查 `agc_wrapper.c` 是否正确编译到 dylib
3. 使用 `nm` 检查符号是否存在

## 八、完整示例代码

查看 `go/example/main_with_dylib.go` 获取完整示例，其中包含：
- 详细的注释说明
- dylib 使用流程
- 错误处理示例

## 九、总结

1. **dylib 位置**：`build/libagc.dylib`
2. **CGO 配置**：在 `go/agc/agc.go` 中通过 `#cgo LDFLAGS` 指定
3. **自动链接**：编译时自动链接，运行时自动加载
4. **路径设置**：使用相对路径 `${SRCDIR}` 确保可移植性
5. **无需手动操作**：Go 代码中无需手动加载 dylib，CGO 自动处理

## 十、快速测试

```bash
# 1. 构建 dylib
./build_so.sh

# 2. 编译 Go 程序
cd go/example
go build

# 3. 运行（会自动加载 dylib）
./example

# 4. 验证链接
otool -L example
```

