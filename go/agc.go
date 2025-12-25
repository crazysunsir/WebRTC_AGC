package agc

/*
#cgo CFLAGS: -I${SRCDIR}/../..
#cgo LDFLAGS: -L${SRCDIR}/../../build -lagc -lm -Wl,-rpath,${SRCDIR}/../../build
#include "agc_wrapper.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"unsafe"
)

// AgcMode 增益模式
type AgcMode int16

const (
	ModeUnchanged       AgcMode = C.AGC_MODE_UNCHANGED        // 0 - 不变模式
	ModeAdaptiveAnalog  AgcMode = C.AGC_MODE_ADAPTIVE_ANALOG  // 1 - 自适应模拟增益
	ModeAdaptiveDigital AgcMode = C.AGC_MODE_ADAPTIVE_DIGITAL // 2 - 自适应数字增益（推荐）
	ModeFixedDigital    AgcMode = C.AGC_MODE_FIXED_DIGITAL    // 3 - 固定数字增益
)

// Config AGC 配置
type Config struct {
	CompressionGaindB int16  // 压缩增益 (dB)，默认 9
	TargetLevelDbfs   int16  // 目标电平 (0-31)，默认 3 (-3 dBOv)
	LimiterEnable     bool   // 限幅器开关，默认 true
	Mode              AgcMode // 增益模式，默认 ModeAdaptiveDigital
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	cConfig := C.AgcGetDefaultConfig()
	return Config{
		CompressionGaindB: int16(cConfig.compressionGaindB),
		TargetLevelDbfs:   int16(cConfig.targetLevelDbfs),
		LimiterEnable:     cConfig.limiterEnable != 0,
		Mode:              AgcMode(cConfig.agcMode),
	}
}

// AGC 自动增益控制实例
type AGC struct {
	instance   unsafe.Pointer
	sampleRate uint32
	config     Config
}

// NewAGC 创建新的 AGC 实例
// sampleRate: 采样率 (8000, 16000, 32000, 48000)
// config: AGC 配置，如果为 nil 则使用默认配置
func NewAGC(sampleRate uint32, config *Config) (*AGC, error) {
	instance := C.AgcCreate()
	if instance == nil {
		return nil, errors.New("failed to create AGC instance")
	}

	// 使用默认配置或提供的配置
	cfg := DefaultConfig()
	if config != nil {
		cfg = *config
	}

	// 初始化 AGC
	minLevel := C.int32_t(0)
	maxLevel := C.int32_t(255)
	ret := C.AgcInit(instance, minLevel, maxLevel, C.int16_t(cfg.Mode), C.uint32_t(sampleRate))
	if ret != 0 {
		C.AgcFree(instance)
		return nil, errors.New("failed to initialize AGC")
	}

	// 设置配置
	cConfig := C.AgcConfig{
		compressionGaindB: C.int16_t(cfg.CompressionGaindB),
		targetLevelDbfs:   C.int16_t(cfg.TargetLevelDbfs),
		limiterEnable:     C.uint8_t(0),
		agcMode:           C.int16_t(cfg.Mode),
	}
	if cfg.LimiterEnable {
		cConfig.limiterEnable = C.uint8_t(1)
	}

	ret = C.AgcSetConfig(instance, &cConfig)
	if ret != 0 {
		C.AgcFree(instance)
		return nil, errors.New("failed to set AGC config")
	}

	return &AGC{
		instance:   instance,
		sampleRate: sampleRate,
		config:     cfg,
	}, nil
}

// Process 处理音频数据（16-bit PCM）
// samples: 输入/输出音频数据，会被原地修改
// 返回: 错误信息
func (a *AGC) Process(samples []int16) error {
	if a.instance == nil {
		return errors.New("AGC instance is nil")
	}
	if len(samples) == 0 {
		return errors.New("samples is empty")
	}

	ret := C.AgcProcess(a.instance, (*C.int16_t)(unsafe.Pointer(&samples[0])),
		C.size_t(len(samples)), C.uint32_t(a.sampleRate))
	if ret != 0 {
		return errors.New("AGC process failed")
	}

	return nil
}

// Close 释放 AGC 实例
func (a *AGC) Close() {
	if a.instance != nil {
		C.AgcFree(a.instance)
		a.instance = nil
	}
}

// GetConfig 获取当前配置
func (a *AGC) GetConfig() Config {
	return a.config
}

// GetSampleRate 获取采样率
func (a *AGC) GetSampleRate() uint32 {
	return a.sampleRate
}

