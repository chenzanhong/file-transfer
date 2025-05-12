package transfer

import (
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"time"

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
func CheckServerBelongsToCompany(username, server string) (bool, error) {
	// conn, err := grpc.Dial("localhost:9000", grpc.WithInsecure())
	// if err != nil {
	// 	logx.Fatalf("did not connect: %v", err)
	// 	return false, err
	// }
	// defer conn.Close()

	// userServiceClient := pb.NewUserServiceClient(conn)
	// hostInfoServiceClient := pb.NewHostInfoServiceClient(conn)

	// // 调用 GetUserByName 获取用户信息
	// userResp, err := userServiceClient.GetUserByName(context.Background(), &pb.GetUserRequest{Name: username})
	// if err != nil {
	// 	logx.Printf("查询用户失败: %v", err)
	// 	return false, err
	// }

	// // 调用 GetHostInfoByName 获取服务器信息
	// hostInfoResp, err := hostInfoServiceClient.GetHostInfoByName(context.Background(), &pb.GetHostInfoRequest{HostName: server})
	// if err != nil {
	// 	logx.Printf("查询服务器失败: %v", err)
	// 	return false, err
	// }

	// // 判断服务器是否属于用户所在的公司或者属于用户自己
	// if userResp.Id == hostInfoResp.CompanyId || userResp.Name == hostInfoResp.UserName {
	// 	return true, nil // 服务器属于用户(所在公司)
	// } else {
	// 	return false, nil // 服务器不属于用户(所在公司)
	// }

	return true, nil // 测试环境,暂时不做判断
}

// 指定两个服务器之间进行单文件传输
func TransferBetweenTwoServer(c *gin.Context) {
	username, exists := c.Get("username") // 从上下文中获取用户名
	if !exists {
		logx.Error("用户未登录")
		c.JSON(401, gin.H{"message": "未登录"})
		return
	}

	var request RequestP2P
	if err := c.BindJSON(&request); err != nil {
		logx.Errorf("解析请求失败: %v", err)
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	flag, err := CheckServerBelongsToCompany(username.(string), request.SourceServer)
	if err != nil {
		logx.Errorf("查询用户与源服务器是否属于同一公司失败: %v", err)
		c.JSON(500, gin.H{"message": fmt.Sprintf("查询用户与源服务器是否属于同一公司失败: %v", err)})
		return
	}
	if !flag {
		logx.Error("该源服务器不是用户所在公司的服务器")
		c.JSON(400, gin.H{"message": "该源服务器不是用户所在公司的服务器"})
		return
	}
	flag, err = CheckServerBelongsToCompany(username.(string), request.TargetServer)
	if err != nil {
		logx.Errorf("查询用户与目的服务器是否属于同一公司失败: %v", err)
		c.JSON(500, gin.H{"message": fmt.Sprintf("查询用户与目的服务器是否属于同一公司失败: %v", err)})
		return
	}
	if !flag {
		logx.Error("该目标服务器不是用户所在公司的服务器")
		c.JSON(400, gin.H{"message": "该目标服务器不是用户所在公司的服务器"})
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
			c.JSON(400, gin.H{"message": fmt.Sprintf("创建与源服务器的连接失败: %v", err)})
			return
		}
	}
	// 检查是否已存在到目标服务器的SSH连接
	if g.FTS.Pool.Connections[request.TargetServer] == nil {
		// 如果不存在，则创建并添加到池中
		err = trans.CreateConnectionToPool(g.Pool, request.TargetServer, request.TargetUser, request.TargetAuth)
		if err != nil {
			logx.Errorf("创建与目标服务器的连接失败: %v", err)
			c.JSON(400, gin.H{"message": fmt.Sprintf("创建与目标服务器的连接失败: %v", err)})
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
		c.JSON(500, gin.H{"message": fmt.Sprintf("文件传输失败: %v", err)})
		return
	}

	logx.Infof("文件传输任务已启动，任务ID: %s", taskID)
}

// 客户端与一个指定的服务器进行文件传输，上传
func CommonUpload(c *gin.Context) {
	username, exists := c.Get("username") // 从上下文中获取用户名
	if !exists {
		logx.Error("用户未登录")
		c.JSON(401, gin.H{"message": "未登录"})
		return
	}

	var request CommonTransRequest
	if err := c.ShouldBind(&request); err != nil {
		logx.Errorf("解析请求失败: %v", err)
		c.JSON(400, gin.H{"message": fmt.Sprintf("解析请求失败: %v", err)})
		return
	}

	// 检查服务器是否属于用户所在的公司或是否是用户自己的服务器
	flag, err := CheckServerBelongsToCompany(username.(string), request.Server)
	if err != nil {
		logx.Errorf("查询服务器与用户（所在公司）的关系失败: %v", err)
		c.JSON(500, gin.H{"message": fmt.Sprintf("查询服务器与用户（所在公司）的关系失败: %v", err)})
		return
	}
	if !flag {
		logx.Error("该服务器不属于用户（所在公司）")
		c.JSON(400, gin.H{"message": "该服务器不属于用户（所在公司）"})
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		logx.Errorf("获取要上传的文件失败: %v", err)
		c.JSON(400, gin.H{"message": fmt.Sprintf("获取要上传的文件失败: %v", err)})
		return
	}

	// 检查是否已存在到指定服务器的SSH连接
	if g.FTS.Pool.Connections[request.Server] == nil {
		// 如果不存在，则创建并添加到池中
		err = trans.CreateConnectionToPool(g.Pool, request.Server, request.User, request.Auth)
		if err != nil {
			logx.Errorf("创建与目标服务器的连接失败: %v", err)
			c.JSON(400, gin.H{"message": fmt.Sprintf("创建与目标服务器的连接失败: %v", err)})
			return
		}
	}

	// 执行文件传输任务
	_, err = g.FTS.CreateCommonUploadTask(
		file,
		request.Server, // 目标服务器IP
		request.Path,   // 目标文件路径
	)
	if err != nil {
		logx.Errorf("文件上传失败: %v", err)
		c.JSON(400, gin.H{"message": fmt.Sprintf("文件上传失败: %v", err)})
		return
	}

	c.JSON(200, gin.H{"message": "文件上传完成"})
}

// 客户端与一个指定的服务器进行文件传输，下载
func CommonDownload(c *gin.Context) {
	username, exists := c.Get("username") // 从上下文中获取用户名
	if !exists {
		logx.Error("用户未登录")
		c.JSON(401, gin.H{"message": "未登录"})
		return
	}
	var request CommonTransRequest
	if err := c.BindJSON(&request); err != nil {
		logx.Errorf("解析请求失败: %v", err)
		c.JSON(400, gin.H{"message": fmt.Sprintf("解析请求失败: %v", err)})
		return
	}

	flag, err := CheckServerBelongsToCompany(username.(string), request.Server)
	if err != nil {
		logx.Errorf("查询服务器与用户（所在公司）的关系失败: %v", err)
		c.JSON(500, gin.H{"message": fmt.Sprintf("查询服务器与用户（所在公司）的关系失败: %v", err)})
		return
	}
	if !flag {
		logx.Stat("该服务器不属于用户（所在公司）")
		c.JSON(400, gin.H{"message": "该服务器不属于用户（所在公司）"})
		return
	}
	// 检查是否已存在到指定服务器的SSH连接
	if g.FTS.Pool.Connections[request.Server] == nil {
		// 如果不存在，则创建并添加到池中
		err = trans.CreateConnectionToPool(g.Pool, request.Server, request.User, request.Auth)
		if err != nil {
			logx.Errorf("创建与目标服务器的连接失败: %v", err)
			c.JSON(400, gin.H{"message": fmt.Sprintf("创建与目标服务器的连接失败: %v", err)})
			return
		}
	}
	// 执行文件传输任务
	sftpClient, _, err := g.FTS.CreateCommonDownloadTask(
		request.Server,
		request.Path,
	)
	if err != nil {
		logx.Errorf("获取连接失败: %v", err)
		c.JSON(400, gin.H{"message": fmt.Sprintf("获取连接失败: %v", err)})
		return
	}
	defer sftpClient.Close()

	file, err := sftpClient.Open(request.Path) // 打开远程文件
	if err != nil {
		logx.Errorf("远程文件打开失败: %v", err)
		c.JSON(400, gin.H{"message": fmt.Sprintf("远程文件打开失败: %v", err)})
		return
	}
	defer file.Close()
	// 判断文件是否存在或是目录
	stat, err := file.Stat() // 获取文件信息，包括大小等
	if err != nil {
		logx.Errorf("文件不存在: %v", err)
		c.JSON(400, gin.H{"message": fmt.Sprintf("文件不存在: %v", err)})
		return
	}
	if stat.IsDir() {
		logx.Errorf("路径是一个目录: %v", err)
		c.JSON(400, gin.H{"message": fmt.Sprintf("路径是一个目录: %v", err)})
		return
	}

	filename := path.Base(request.Path)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	fi, err := file.Stat()
	if err != nil {
		logx.Errorf("获取文件信息失败: %v", err)
		c.JSON(500, gin.H{"message": "获取文件信息失败"})
		return
	}
	c.Header("Content-Length", strconv.FormatInt(fi.Size(), 10))

	c.Writer.WriteHeader(200)

	if _, err := io.Copy(c.Writer, file); err != nil {
		if strings.Contains(err.Error(), "broken pipe") || err.Error() == "connection lost" {
			logx.Error("客户端已断开连接")
			return
		}
		logx.Errorf("文件写入响应失败: %v", err)
		return
	}
	c.Writer.Flush()
}
