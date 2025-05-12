package transfer

import (
	"file-transfer/transfer/global"
	trans "file-transfer/transfer/trans-init"
	"io"
)

// UploadFileToServer 将文件内容上传到目标服务器
func UploadFileToServer(server, path, user, auth string, fileData []byte) error {
	// 如果不存在连接，尝试创建
	if global.FTS.Pool.Connections[server] == nil {
		err := trans.CreateConnectionToPool(global.Pool, server, user, auth)
		if err != nil {
			return err
		}
	}

	// 创建上传任务
	_, err := global.FTS.CreateCommonUploadTaskFromBytes(fileData, server, path)
	return err
}

// DownloadFileFromServer 从服务器下载文件并返回字节流
func DownloadFileFromServer(server, path, user, auth string) ([]byte, error) {
	if global.FTS.Pool.Connections[server] == nil {
		err := trans.CreateConnectionToPool(global.Pool, server, user, auth)
		if err != nil {
			return nil, err
		}
	}

	sftpClient, _, err := global.FTS.CreateCommonDownloadTask(server, path)
	if err != nil {
		return nil, err
	}
	defer sftpClient.Close()

	file, err := sftpClient.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

// TransferBetweenTwoServers 实现两个服务器之间的文件传输
func TransferBetweenTwoServers(srcServer, srcPath, destServer, destPath string,
	srcUser, dstUser, srcAuth, dstAuth string) error {

	if global.FTS.Pool.Connections[srcServer] == nil {
		if err := trans.CreateConnectionToPool(global.Pool, srcServer, srcUser, srcAuth); err != nil {
			return err
		}
	}
	if global.FTS.Pool.Connections[destServer] == nil {
		if err := trans.CreateConnectionToPool(global.Pool, destServer, dstUser, dstAuth); err != nil {
			return err
		}
	}

	_, err := global.FTS.CreateTransferBetween2STask(srcServer, srcPath, destServer, destPath)
	return err
}