package main

/*
这个示例展示了如何在 Go 代码中直接使用编译好的 dylib 文件

前提条件：
1. 已编译生成 dylib 文件：./build_so.sh
2. dylib 文件位置：build/libagc.dylib
3. 头文件位置：go/agc/agc_wrapper.h

CGO 会自动：
- 通过 LDFLAGS 中的 -L 参数找到 dylib 文件
- 通过 -lagc 链接 libagc.dylib
- 通过 -Wl,-rpath 设置运行时搜索路径
*/

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"webrtc-agc/agc" // 导入 AGC 包，CGO 会自动链接 dylib
)

func main() {
	fmt.Println("=== Go 中使用 dylib 示例 ===")
	fmt.Println("")
	fmt.Println("说明：")
	fmt.Println("1. dylib 文件位置: build/libagc.dylib")
	fmt.Println("2. CGO 通过 LDFLAGS 自动链接: -L${SRCDIR}/../../build -lagc")
	fmt.Println("3. 运行时路径: -Wl,-rpath,${SRCDIR}/../../build")
	fmt.Println("")

	// 示例：处理 WAV 文件
	inputFile := "/Users/sunxuguang/WebRTC_AGC/通话记录-1725189.wav"
	outputFile := "/Users/sunxuguang/WebRTC_AGC/通话记录-1725189_go.wav"

	// 读取 WAV 文件
	samples, sampleRate, err := readWAVFile(inputFile)
	if err != nil {
		fmt.Printf("读取文件失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ 读取文件成功\n")
	fmt.Printf("  采样率: %d Hz\n", sampleRate)
	fmt.Printf("  采样点数: %d\n", len(samples))
	fmt.Println("")

	// 创建 AGC 配置
	// 这些参数会传递给 dylib 中的 C 函数
	config := &agc.Config{
		CompressionGaindB: 12,              // 压缩增益 12 dB
		TargetLevelDbfs:   1,               // 目标电平 -1 dBOv
		LimiterEnable:     true,            // 开启限幅器
		Mode:              agc.ModeAdaptiveDigital, // 自适应数字模式
	}

	fmt.Println("创建 AGC 实例（调用 dylib 中的函数）...")
	// NewAGC 内部会调用 dylib 中的 AgcCreate() 和 AgcInit()
	agcInstance, err := agc.NewAGC(sampleRate, config)
	if err != nil {
		fmt.Printf("✗ 创建 AGC 实例失败: %v\n", err)
		fmt.Println("  提示: 请确保 build/libagc.dylib 文件存在")
		os.Exit(1)
	}
	defer agcInstance.Close() // 释放 dylib 中的资源

	fmt.Println("✓ AGC 实例创建成功（dylib 已加载）")
	fmt.Println("")

	// 处理音频数据
	// Process 内部会调用 dylib 中的 AgcProcess()
	fmt.Println("处理音频数据（调用 dylib 中的 AgcProcess）...")
	err = agcInstance.Process(samples)
	if err != nil {
		fmt.Printf("✗ 处理音频失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ 音频处理完成")
	fmt.Println("")

	// 保存处理后的音频
	err = writeWAVFile(outputFile, samples, sampleRate)
	if err != nil {
		fmt.Printf("保存文件失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ 输出文件已保存: %s\n", outputFile)
	fmt.Println("")
	fmt.Println("=== 处理完成 ===")
	fmt.Println("")
	fmt.Println("说明：")
	fmt.Println("- dylib 文件在编译时自动链接")
	fmt.Println("- 运行时通过 rpath 自动找到 dylib")
	fmt.Println("- 所有 AGC 操作都通过 dylib 中的 C 函数完成")
}

// 读取 WAV 文件的 PCM 数据
func readWAVFile(filename string) ([]int16, uint32, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	// 读取 WAV 头
	header := make([]byte, 44)
	_, err = io.ReadFull(file, header)
	if err != nil {
		return nil, 0, err
	}

	// 解析采样率（WAV 文件第 24-27 字节）
	sampleRate := binary.LittleEndian.Uint32(header[24:28])

	// 读取 PCM 数据
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, 0, err
	}

	// 转换为 int16 数组
	samples := make([]int16, len(data)/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
	}

	return samples, sampleRate, nil
}

// 写入 WAV 文件
func writeWAVFile(filename string, samples []int16, sampleRate uint32) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// 写入 WAV 头
	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(36+len(samples)*2))
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16) // fmt chunk size
	binary.LittleEndian.PutUint16(header[20:22], 1)  // audio format (PCM)
	binary.LittleEndian.PutUint16(header[22:24], 1)  // num channels
	binary.LittleEndian.PutUint32(header[24:28], sampleRate)
	binary.LittleEndian.PutUint32(header[28:32], sampleRate*2) // byte rate
	binary.LittleEndian.PutUint16(header[32:34], 2)            // block align
	binary.LittleEndian.PutUint16(header[34:36], 16)           // bits per sample
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], uint32(len(samples)*2))

	_, err = file.Write(header)
	if err != nil {
		return err
	}

	// 写入 PCM 数据
	for _, sample := range samples {
		binary.Write(file, binary.LittleEndian, sample)
	}

	return nil
}

