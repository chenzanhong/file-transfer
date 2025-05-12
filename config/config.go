package config

import (
	"os"
	"path/filepath"
	"runtime"
	// "fmt"

	"github.com/zeromicro/go-zero/core/logx"
	"gopkg.in/yaml.v2"
)

// LogxConfig 对应 YAML 中 Logger 的配置项
type LogxConfig struct {
	ServiceName string `yaml:"ServiceName"`
	Mode        string `yaml:"Mode"`       // file/console
	Encoding    string `yaml:"Encoding"`   // plain/json
	Level       string `yaml:"Level"`      // debug/info/warn/error/fatal
	Path        string `yaml:"Path"`       // 日志路径（当 Mode 为 file 时使用）
	Stat		bool   `yaml:"Stat"`
	KeepDays    int    `yaml:"KeepDays"`   // 保留天数
	MaxBackups  int    `yaml:"MaxBackups"` // 最多保留旧日志文件个数
	MaxSize     int    `yaml:"MaxSize"`    // 每个日志文件最大 MB
	Compress    bool   `yaml:"Compress"`   // 是否压缩日志
}

// Config 用于保存所有配置项
type Config struct {
	Logger LogxConfig `yaml:"Logger"`
}

// getConfigPath 获取配置文件的路径
func getConfigPath() string {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		logx.Error("无法获取运行时调用者信息")
		return ""
	}

	currentDir := filepath.Dir(filename)
	configPath := filepath.Join(currentDir, "..", "config", "config", "config.yaml")

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		logx.Errorf("无法获取绝对路径: %v", err)
	}

	simplifiedPath := filepath.Clean(absPath)
	return simplifiedPath
}

// GetConfigPath 返回配置文件的路径
func GetConfigPath() string {
	return getConfigPath()
}

// LoadConfig 加载配置文件并返回 Config
func LoadConfig() (*Config, error) {
	configPath := GetConfigPath()
	yamlFile, err := os.ReadFile(configPath)
	if err != nil {
		logx.Errorf("读取配置文件失败: %v", err)
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		logx.Errorf("解析配置文件失败: %v", err)
		return nil, err
	}
	
	// 打印加载的配置
	// fmt.Printf("加载的配置: %+v\n", config)

	return &config, nil
}

// SetupLogx 使用 go-zero 的 logx 初始化日志系统
func SetupLogx(cfg *Config) {
	logConf := logx.LogConf{
		ServiceName: cfg.Logger.ServiceName,
		Mode:        cfg.Logger.Mode,
		Encoding:    cfg.Logger.Encoding,
		Level:       cfg.Logger.Level,
		Path:        cfg.Logger.Path,
		Stat:		 cfg.Logger.Stat,
		KeepDays:    cfg.Logger.KeepDays,
		MaxBackups:  cfg.Logger.MaxBackups,
		MaxSize:     cfg.Logger.MaxSize,
		Compress:    cfg.Logger.Compress,
	}

	logx.SetUp(logConf)
}
