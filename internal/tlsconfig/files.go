package tlsconfig

import (
	"os"
	"path/filepath"
)

// 证书和私钥的绝对路径名
// 后续创建服务器和客户端时会使用这些路径下的证书和私钥
var (
	CAFile         = configFile("ca.pem")
	ServerCertFile = configFile("server.pem")
	ServerKeyFile  = configFile("server-key.pem")
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
