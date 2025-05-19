package main

import (
	"file-transfer/middlewire"
	cors "file-transfer/middlewire/cors"

	"file-transfer/config"
	grpcserver "file-transfer/grpc"
	ft "file-transfer/proto/file-transfer"
	"file-transfer/transfer"
	g "file-transfer/transfer/global"
	trans "file-transfer/transfer/trans-init"

	"net"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		logx.Errorf("加载配置失败：%v", err)
	}

	config.SetupLogx(cfg)

	router := gin.Default()
	router.Use(cors.CORSMiddleware())

	// 初始化SSH连接池及文件传输服务
	g.Pool = trans.NewSSHConnectionPool(10, 5*time.Minute)
	stopChan := make(chan struct{})
	defer close(stopChan)
	go g.Pool.Cleanup(stopChan)          // 启动清理协程
	trans.NewFileTransferService(g.Pool) // 初始化文件传输服务
	g.FTS = trans.NewFileTransferService(g.Pool)

	// go monitor.CheckServerStatus()
	router.Static("/static", "./static")

	// 需要 JWT 认证的路由
	auth := router.Group("/agent", middlewire.JWTAuthMiddleware())
	{
		// 文件传输
		auth.POST("/upload", transfer.CommonUpload)
		auth.POST("/download", transfer.CommonDownload)
		auth.POST("/transfer", transfer.TransferBetweenTwoServer)
	}

	// 启动 gRPC 服务
	go func() {
		lis, err := net.Listen("tcp", ":9002")
		if err != nil {
			logx.Errorf("failed to listen: %v", err)
		}
		grpcServer := grpc.NewServer()
		ft.RegisterFileTransferServiceServer(grpcServer, &grpcserver.Server{})
		logx.Info("gRPC 服务正在监听：9002")
		if err := grpcServer.Serve(lis); err != nil {
			logx.Errorf("failed to serve : %v", err)
		}
	}()

	router.Run("0.0.0.0:9085")
}
