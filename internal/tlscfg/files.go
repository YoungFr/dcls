package tlscfg

import (
	"os"
	"path/filepath"
)

// 证书和私钥的绝对路径名
// 后续创建服务器和客户端时会使用这些路径下的证书和私钥
var (
	CAFile = configFile("ca.pem")

	// 服务端
	ServerCertFile = configFile("server.pem")
	ServerKeyFile  = configFile("server-key.pem")

	// 超级用户
	RootClientCertFile = configFile("root-client.pem")
	RootClientKeyFile  = configFile("root-client-key.pem")

	// 普通用户
	OrdinaryClientCertFile = configFile("ordinary-client.pem")
	OrdinaryClientKeyFile  = configFile("ordinary-client-key.pem")

	// 只读用户
	ReadOnlyClientCertFile = configFile("readonly-client.pem")
	ReadOnlyClientKeyFile  = configFile("readonly-client-key.pem")

	// 配置和策略文件
	ACLModelFile  = configFile("model.conf")
	ACLPolicyFile = configFile("policy.csv")
)

func configFile(filename string) string {
	if dir := os.Getenv("CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, filename)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(homeDir, ".dcls", filename)
}
