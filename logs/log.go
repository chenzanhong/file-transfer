package logs

import (
	"bufio"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zeromicro/go-zero/core/logx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Sugar *zap.SugaredLogger

func InitZapSugarDefault() {
	_, err := os.OpenFile("./logs/user_operation_log/user_operation_log.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logx.Errorf("创建/打开日志文件失败：%v", err)
		return
	}
	// 配置日志文件
	fileLooger := &lumberjack.Logger{
		Filename:   "./logs/user_operation_log/user_operation_log.log",
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     60,
		Compress:   false,
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(fileLooger),
		zapcore.InfoLevel,
	)

	// logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	// 用户操作日志，不需要调用信息和堆栈信息
	logger := zap.New(core)

	Sugar = logger.Sugar()
}

func SetupZapSugar(path string, level zapcore.Level) {
	fileLooger := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     60,
		Compress:   false,
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(fileLooger),
		level,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	Sugar = logger.Sugar()
}

// 获取用户操作日志
func GetUserOperationLog(c *gin.Context) {
	Username, exists := c.Get("username")
	if !exists {
		logx.Errorf("获取用户信息失败")
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "获取用户信息失败",
		})
		return
	}
	username := Username.(string)

	// 打开日志文件
	file, err := os.OpenFile("./logs/user_operation_log/user_operation_log.log", os.O_RDONLY, 0644)
	if err != nil {
		logx.Errorf("打开日志文件失败：%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "打开日志文件失败",
		})
		return
	}
	defer file.Close()

	var response []string

	// 读取日志文件内容
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// 检查日志中是否包含指定的用户名
		if strings.Contains(line, "\"username\":\""+username+"\"") {
			response = append(response, line)
		}
	}
	if err := scanner.Err(); err != nil {
		logx.Errorf("读取日志文件失败：%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "读取日志文件失败",
		})
		return
	}
	// 返回日志内容给前端
	c.JSON(200, gin.H{
		"message": "success",
		"logs":    response,
	})
}
