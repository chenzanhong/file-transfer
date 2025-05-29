package transfer

import (
	// "context"

	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"file-transfer/logs"
	g "file-transfer/transfer/global"
	trans "file-transfer/transfer/trans-init" // 请替换为您的实际项目路径

	"github.com/gin-gonic/gin"
	"github.com/zeromicro/go-zero/core/logx"
)

type RequestP2P struct {
	SourceServer string `json:"source_server"`
	TargetServer string `json:"target_server"`
	SourcePath   string `json:"source_path"`
	TargetPath   string `json:"target_path"`
	SourceUser   string `json:"source_user"`
	TargetUser   string `json:"target_user"`
	SourceAuth   string `json:"source_auth"`
	TargetAuth   string `json:"target_auth"`
}

type CommonTransRequest struct {
	Server string `json:"server" form:"server"` // 服务器地址
	Path   string `json:"path" form:"path"`     // 文件路径
	User   string `json:"user" form:"user"`     // SSH用户名
	Auth   string `json:"auth" form:"auth"`     // SSH密码或密钥
}

// 查询服务器是否是用户所在公司的服务器
func CheckServerBelongs(username, server string) (bool, error) {
	// conn, err := grpc.NewClient("localhost:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	// if err != nil {
	// 	logx.Errorf("did not connect: %v", err)
	// 	return false, err
	// }
	// defer conn.Close()

	// userServiceClient := user.NewUserServiceClient(conn)

	// // 调用 GetUserByName 获取用户信息
	// userResp, err := userServiceClient.GetUserInfo(context.Background(), &user.GetUserInfoRequest{Username: username})
	// if err != nil {
	// 	logx.Errorf("查询用户失败: %v", err)
	// 	return false, err
	// }

	// // 调用 GetHostInfoByName 获取服务器信息
	// hostInfoResp, err := userServiceClient.GetHostInfo(context.Background(), &user.GetHostInfoRequest{Hostname: server})
	// if err != nil {
	// 	logx.Errorf("查询服务器失败: %v", err)
	// 	return false, err
	// }

	// // 判断服务器是否属于用户所在的公司或者属于用户自己
	// if userResp.CompanyId == hostInfoResp.CompanyId || userResp.Name == hostInfoResp.UserName {
	// 	return true, nil // 服务器属于用户(所在公司)
	// } else {
	// 	return false, nil // 服务器不属于用户(所在公司)
	// }

	return true, nil // 测试环境,暂时不做判断
}

// 指定两个服务器之间进行单文件传输
func TransferBetweenTwoServer(c *gin.Context) {
	Username, exists := c.Get("username") // 从上下文中获取用户名
	if !exists {
		logx.Error("用户未登录")
		c.JSON(http.StatusUnauthorized, gin.H{"message": "未登录"})
		return
	}
	username := Username.(string)

	var request RequestP2P
	if err := c.BindJSON(&request); err != nil {
		logx.Errorf("解析请求失败: %v", err)
		logs.Sugar.Errorw("指定两个服务器之间进行单文件传输", "username", username, "detail", "解析请求失败，请检查请求格式是否正确")
		c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("解析请求失败: %v", err)})
		return
	}

	flag, err := CheckServerBelongs(username, request.SourceServer)
	if err != nil {
		logx.Errorf("查询源服务器是否属于用户（所在公司）: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("查询源服务器是否属于用户（所在公司）失败: %v", err.Error())})
		return
	}
	if !flag {
		logx.Error("该源服务器不属于用户（所在公司）")
		logs.Sugar.Errorw("指定两个服务器之间进行单文件传输", "username", username, "detail", "该源服务器不属于用户（所在公司）")
		c.JSON(http.StatusBadRequest, gin.H{"message": "该源服务器不属于用户（所在公司）"})
		return
	}
	flag, err = CheckServerBelongs(username, request.TargetServer)
	if err != nil {
		logx.Errorf("查询用户与目标服务器是否属于同一公司失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("查询目标服务器是否属于用户（所在公司）失败: %v", err.Error())})
		return
	}
	if !flag {
		logx.Error("该目标服务器不属于用户（所在公司）")
		logs.Sugar.Errorw("指定两个服务器之间进行单文件传输", "username", username, "detail", "该目标服务器不属于用户（所在公司）")
		c.JSON(http.StatusForbidden, gin.H{"message": "该目标服务器不属于用户（所在公司）"})
		return
	}

	// 判断是否存在连接池，如果不存在则创建
	if g.Pool == nil {
		g.Pool = trans.NewSSHConnectionPool(10, 5*time.Minute) // 假设容量为10，超时时间为5分钟
	}
	// 检查是否已存在到源服务器的SSH连接
	if g.FTS.Pool.Connections[request.SourceServer] == nil {
		// 如果不存在，则创建并添加到池中
		err = trans.CreateConnectionToPool(g.Pool, request.SourceServer, request.SourceUser, request.SourceAuth)
		if err != nil {
			logx.Errorf("创建与源服务器的连接失败: %v", err)
			logs.Sugar.Errorw("指定两个服务器之间进行单文件传输", "username", username, "detail", "创建与源服务器的连接失败，请检查源服务器是否正确")
			c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("创建与源服务器的连接失败: %v", err)})
			return
		}
	}
	// 检查是否已存在到目标服务器的SSH连接
	if g.FTS.Pool.Connections[request.TargetServer] == nil {
		// 如果不存在，则创建并添加到池中
		err = trans.CreateConnectionToPool(g.Pool, request.TargetServer, request.TargetUser, request.TargetAuth)
		if err != nil {
			logx.Errorf("创建与目标服务器的连接失败: %v", err)
			logs.Sugar.Errorw("指定两个服务器之间进行单文件传输", "username", username, "detail", "创建与目标服务器的连接失败，请检查目标服务器是否正确")
			c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("创建与目标服务器的连接失败: %v", err)})
			return
		}
	}

	// 执行文件传输任务
	taskID, err := g.FTS.CreateTransferBetween2STask(
		request.SourceServer, // 源服务器IP
		request.SourcePath,   // 源文件路径
		request.TargetServer, // 目标服务器IP
		request.TargetPath,   // 目标文件路径
	)
	if err != nil {
		logx.Errorf("文件传输失败: %v", err)
		logs.Sugar.Errorw("指定两个服务器之间进行单文件传输", "username", username, "detail", "文件传输失败，请确认文件路径是否正确")
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("文件传输失败: %v", err), "task_id": taskID})
		return
	}

	logx.Infof("文件传输成功，任务ID: %s", taskID)
	logs.Sugar.Infow("指定两个服务器之间进行单文件传输", "username", username, "detail", "文件传输成功，任务ID："+taskID)
	c.JSON(http.StatusOK, gin.H{"message": "文件传输任务已启动", "task_id": taskID})
}

// 客户端与一个指定的服务器进行文件传输，上传
func CommonUpload(c *gin.Context) {
	Username, exists := c.Get("username") // 从上下文中获取用户名
	if !exists {
		logx.Error("用户未登录")
		c.JSON(http.StatusUnauthorized, gin.H{"message": "未登录"})
		return
	}
	username := Username.(string)

	var request CommonTransRequest
	if err := c.ShouldBind(&request); err != nil {
		logx.Errorf("解析请求失败: %v", err)
		logs.Sugar.Errorw("文件上传", "username", username, "detail", "解析请求失败，请检查请求格式是否正确")
		c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("解析请求失败: %v", err)})
		return
	}

	// 检查服务器是否属于用户所在的公司或是否是用户自己的服务器
	flag, err := CheckServerBelongs(username, request.Server)
	if err != nil {
		logx.Errorf("查询服务器与用户（所在公司）的关系失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("查询服务器与用户（所在公司）的关系失败: %v", err)})
		return
	}
	if !flag {
		logx.Error("该服务器不属于用户（所在公司）")
		logs.Sugar.Errorw("文件上传", "username", username, "detail", "该服务器不属于用户（所在公司）")
		c.JSON(http.StatusBadRequest, gin.H{"message": "该服务器不属于用户（所在公司）"})
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		logx.Errorf("获取要上传的文件失败: %v", err)
		logs.Sugar.Errorw("文件上传", "username", username, "detail", "获取要上传的文件失败")
		c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("获取要上传的文件失败: %v", err)})
		return
	}

	// 检查是否已存在到指定服务器的SSH连接
	if g.FTS.Pool.Connections[request.Server] == nil {
		// 如果不存在，则创建并添加到池中
		err = trans.CreateConnectionToPool(g.Pool, request.Server, request.User, request.Auth)
		if err != nil {
			logx.Errorf("创建与目标服务器的连接失败: %v", err)
			logs.Sugar.Errorw("文件上传", "username", username, "detail", "创建与目标服务器的连接失败，请检查目标服务器是否正确")
			c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("创建与目标服务器的连接失败: %v", err)})
			return
		}
	}

	// 执行文件传输任务
	taskID, err := g.FTS.CreateCommonUploadTask(
		file,
		request.Server, // 目标服务器IP
		request.Path,   // 目标文件路径
	)
	if err != nil {
		logx.Errorf("文件上传失败: %v", err)
		logs.Sugar.Errorw("文件上传", "username", username, "detail", "文件上传失败，请检查文件路径是否正确")
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("文件上传失败: %v", err), "task_id": taskID})
		return
	}

	fmt.Printf("文件上传任务已完成，任务ID: %s\n", taskID)
	logs.Sugar.Infow("文件上传", "username", username, "detail", "文件上传成功，任务ID："+taskID)
	c.JSON(http.StatusOK, gin.H{"message": "文件上传完成", "task_id": taskID})
}

// 客户端与一个指定的服务器进行文件传输，下载
func CommonDownload(c *gin.Context) {
	Username, exists := c.Get("username") // 从上下文中获取用户名
	if !exists {
		logx.Error("用户未登录")
		c.JSON(http.StatusUnauthorized, gin.H{"message": "未登录"})
		return
	}
	username := Username.(string)

	var request CommonTransRequest
	if err := c.ShouldBind(&request); err != nil {
		logx.Errorf("解析请求失败: %v", err)
		logs.Sugar.Errorw("文件下载", "username", username, "detail", "解析请求失败，请检查请求格式是否正确")
		c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("解析请求失败: %v", err)})
		return
	}

	flag, err := CheckServerBelongs(username, request.Server)
	if err != nil {
		logx.Errorf("查询服务器与用户（所在公司）的关系失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("查询服务器与用户（所在公司）的关系失败: %v", err)})
		return
	}
	if !flag {
		logx.Error("该服务器不属于用户（所在公司）")
		logs.Sugar.Errorw("文件下载", "username", username, "detail", "该服务器不属于用户（所在公司）")
		c.JSON(http.StatusBadRequest, gin.H{"message": "该服务器不属于用户（所在公司）"})
		return
	}
	// 检查是否已存在到指定服务器的SSH连接
	if g.FTS.Pool.Connections[request.Server] == nil {
		// 如果不存在，则创建并添加到池中
		err = trans.CreateConnectionToPool(g.Pool, request.Server, request.User, request.Auth)
		if err != nil {
			logx.Errorf("创建与目标服务器的连接失败: %v", err)
			logs.Sugar.Errorw("文件下载", "username", username, "detail", "创建与目标服务器的连接失败，请检查目标服务器是否正确")
			c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("创建与目标服务器的连接失败: %v", err)})
			return
		}
	}
	// 执行文件传输任务
	sftpClient, task_id, err := g.FTS.CreateCommonDownloadTask(
		request.Server,
		request.Path,
	)
	if err != nil {
		logx.Errorf("获取连接失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("获取连接失败: %v", err)})
		return
	}
	defer sftpClient.Close()

	file, err := sftpClient.Open(request.Path) // 打开远程文件
	if err != nil {
		logx.Errorf("远程文件打开失败: %v", err)
		logs.Sugar.Errorw("文件下载", "username", username, "detail", "远程文件打开失败，请检查文件路径是否正确")
		c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("远程文件打开失败: %v", err)})
		return
	}
	defer file.Close()

	// 判断文件是否存在或是目录
	stat, err := file.Stat() // 获取文件信息，包括大小等
	if err != nil {
		logx.Errorf("文件不存在: %v", err)
		logs.Sugar.Errorw("文件下载", "username", username, "detail", "文件不存在")
		c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("文件不存在: %v", err)})
		return
	}
	if stat.IsDir() {
		logx.Errorf("路径是一个目录: %v", err)
		logs.Sugar.Errorw("文件下载", "username", username, "detail", "路径是一个目录")
		c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("路径是一个目录: %v", err)})
		return
	}

	filename := path.Base(request.Path)
	encodedFilename := url.PathEscape(filename)
	// fmt.Println(filename + "\n" + encodedFilename)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; "+fmt.Sprintf(`filename="%s"; filename*=UTF-8''%s`,
		encodedFilename, encodedFilename))

	fi, err := file.Stat() // 获取文件信息，包括大小等
	if err != nil {
		logx.Errorf("获取文件信息失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取文件信息失败"})
		return
	}
	c.Header("Content-Length", strconv.FormatInt(fi.Size(), 10)) // 设置文件大小

	// WriterHeader 不是必须的，Gin会自动处理
	// c.Writer.WriteHeader(http.StatusOK)

	if _, err := io.Copy(c.Writer, file); err != nil {
		if strings.Contains(err.Error(), "broken pipe") || err.Error() == "connection lost" {
			logx.Error("客户端已断开连接")
			return
		}
		logx.Errorf("文件写入响应失败: %v", err)
		logs.Sugar.Errorw("文件下载", "username", username, "detail", "文件写入响应失败，请检查网络连接是否正常")
		return
	}
	c.Writer.Flush()
	logs.Sugar.Infow("文件下载", "username", username, "detail", "文件下载成功，任务ID："+task_id)
}
