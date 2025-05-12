package main

import (
	"context"
	"fmt"
	"io/ioutil"

	pb "file-transfer/pb" // 替换为你的实际项目路径

	"google.golang.org/grpc"
	
	"github.com/zeromicro/go-zero/core/logx"
)

func main() {
	// 连接到服务器
	conn, err := grpc.Dial("localhost:9002", grpc.WithInsecure())
	if err != nil {
		logx.Errorf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewFileTransferServiceClient(conn)

	// 准备上传的文件数据
	filePath := "C:/Users/HUAWEI/Desktop/文档.txt" // 更改为你要上传的文件路径
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		logx.Errorf("failed to read file: %v", err)
	}

	// 构建请求
	req := &pb.CommonUploadRequest{
		Server:   "192.168.202.128",          // 目标服务器 IP 或域名
		Path:     "/home/czh/Desktop/文档.txt", // 远程路径
		User:     "root",
		Auth:     "czh_centos",
		FileData: data,
	}

	// 发送请求
	resp, err := client.CommonUpload(context.Background(), req)
	if err != nil {
		logx.Errorf("could not upload file: %v", err)
	}

	fmt.Printf("Response message: %s\n", resp.Message)
}
