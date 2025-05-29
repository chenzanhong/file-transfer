# file-transfer
指定两台服务器间或本地与某台远程服务器间进行单文件传输



// 创建一个集合用于记录已经frp中被使用的remote_port
var usedPorts = make(map[int]bool)

// 生成一个新的可用的remote_port
func getNewRemotePort() int {
	for {
		port := 6000 + len(usedPorts) // 从6000开始递增
		if !usedPorts[port] {
			usedPorts[port] = true // 标记为已使用
			return port
		}
	}
}

// DoInstallAgent 执行 agent 安装
func DoInstallAgent(ss SshInfo) error {
	// 从环境变量中获取 frp 服务地址和端口
	frpServerAddr := os.Getenv("FRP_SERVER_ADDR")
	frpServerPortStr := os.Getenv("FRP_SERVER_PORT")
	var frpServerPort int
	if frpServerPortStr != "" {
		frpServerPort, _ = strconv.Atoi(frpServerPortStr)
	}

	// 如果不是root用户会报权限不足，需要输入密码

	// SSH 配置
	config := &ssh.ClientConfig{
		User: ss.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(ss.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 建立 SSH 连接
	s := fmt.Sprintf("%s:%v", ss.Host, ss.Port)
	client, err := ssh.Dial("tcp", s, config)
	if err != nil {
		fmt.Printf("Failed to dial: %s", err)
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		fmt.Printf("Failed to create session: %s", err)
		return err
	}
	defer session.Close()

	packageCmd := ""
	switch ss.Platform {
	case "ubuntu", "debian":
		packageCmd = "apt update && apt install -y git"
	case "centos", "rhel", "fedora":
		packageCmd = "yum install -y git"
	default:
		return fmt.Errorf("unsupported platform: %s", ss.Platform)
	}

	// 动态生成 frpc.ini 的内容
	frpcIniContent := fmt.Sprintf(`
[common]
server_addr = %s
server_port = %d

[web]
type = tcp
local_ip = 127.0.0.1
local_port = 8080 # 或者你希望转发的服务端口
remote_port = %d # 远程端口,每一个agent应该使用不同的remote_port，动态生成且被监控的服务器上未被占用
`, frpServerAddr, frpServerPort, getNewRemotePort())

	cmd := fmt.Sprintf(
		`#!/bin/bash
%s

# 清理旧的 agent 目录
# rm -rf /opt/agent

# 克隆代码仓库
git clone https://gitee.com/chenzanhong/agent.git /opt/agent
cd /opt/agent || exit

# 检查 main 文件是否存在
if [ ! -f "/opt/agent/main" ]; then
echo "Main executable not found!"
exit 1
fi

# 授予执行权限并运行主程序
chmod +x main
./main -hostname="%s" -token="%s" &

# 创建 frpc.ini 文件
mkdir -p /etc/frp
cat <<EOF > /etc/frp/frpc.ini
%s
EOF

# 创建 systemd 服务文件
cat <<EOF | sudo tee /etc/systemd/system/main_startup.service
[Unit]
Description=Main Program Startup Service
After=network.target

[Service]
Type=simple
ExecStart=/opt/agent/main -hostname=%s -token=%s
Restart=always

[Install]
WantedBy=multi-user.target
EOF

# 创建 frpc systemd 服务文件
cat <<EOF | sudo tee /etc/systemd/system/frpc.service
[Unit]
Description=Frp Client Service
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/frpc -c /etc/frp/frpc.ini
Restart=always

[Install]
WantedBy=multi-user.target
EOF

# 启用并启动服务
# sudo systemctl enable main_startup.service
sudo systemctl start main_startup.service
# sudo systemctl enable frpc.service
sudo systemctl start frpc.service
`,
		packageCmd, ss.Host_Name, ss.Token, frpcIniContent, ss.Host_Name, ss.Token)

	// 运行命令
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		fmt.Printf("Failed to run command: %s\nOutput: %s", err, string(output))
		return err
	}
	fmt.Println(string(output))

	return nil
}


package configs

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Url           string `yaml:"url"`
	Port          string `yaml:"port"`
	Second        int    `yaml:"second"`
	FrpPublicIp   string `yaml:"frp_public_ip"`   // frp
	FrpRemotePort string `yaml:"frp_remote_port"` // frp
}

func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open config file: %v", err)
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("unable to parse config file: %v", err)
	}
	return &config, nil
}


frp_public_ip: 47.86.232.20
frp_remote_port: 7000
second: 30
url:
port: 8080