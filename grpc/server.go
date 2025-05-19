package grpcserver

import (
	"context"

	ft "file-transfer/proto/file-transfer"
	"file-transfer/transfer"

	"github.com/zeromicro/go-zero/core/logx"
)

type Server struct {
	ft.UnimplementedFileTransferServiceServer
}

func (s *Server) CommonUpload(ctx context.Context, req *ft.CommonUploadRequest) (*ft.CommonUploadResponse, error) {
	err := transfer.UploadFileToServer(req.Server, req.Path, req.User, req.Auth, req.FileData)
	if err != nil {
		logx.Errorf("文件上传失败: %v", err)
		return &ft.CommonUploadResponse{Message: "上传失败"}, err
	}
	return &ft.CommonUploadResponse{Message: "上传成功"}, nil
}

func (s *Server) CommonDownload(req *ft.CommonDownloadRequest, stream ft.FileTransferService_CommonDownloadServer) error {
	data, err := transfer.DownloadFileFromServer(req.Server, req.Path, req.User, req.Auth)
	if err != nil {
		return err
	}

	const chunkSize = 1024 * 32 // 32KB per chunk
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}

		if err := stream.Send(&ft.FileChunk{Content: data[i:end]}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) TransferBetweenTwoServers(ctx context.Context, req *ft.TransferBetweenRequest) (*ft.TransferResponse, error) {
	err := transfer.TransferBetweenTwoServers(
		req.SourceServer, req.TargetServer, req.SourcePath, req.TargetPath,
		req.SourceUser, req.TargetUser, req.SourceAuth, req.TargetAuth,
	)
	if err != nil {
		logx.Errorf("文件传输失败: %v", err)
		return &ft.TransferResponse{Message: "传输失败"}, err
	}
	return &ft.TransferResponse{Message: "传输完成"}, nil
}
