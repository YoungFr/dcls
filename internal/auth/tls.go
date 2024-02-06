package auth

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
)

type TLSConfig struct {
	// 值为真表示是服务端 TLS 配置
	// 值为假表示是客户端 TLS 配置
	IsServerConfig bool

	// 是否启用双向 TLS 认证
	EnableMutualTLS bool

	CertFile   string
	KeyFile    string
	CAFile     string
	ServerName string
}

var (
	// server
	errNoCertForServerTLSConfig = errors.New("no certificate provided for server (mutual) TLS config")
	errNoPKeyForServerTLSConfig = errors.New("no private key provided for server (mutual) TLS config")
	errNoCAForServerMuTLSConfig = errors.New("no root certificate provided for server mutual TLS config")

	// client
	errNoCertForClientMuTLSConfig = errors.New("no certificate provided for client mutual TLS config")
	errNoPKeyForClientMuTLSConfig = errors.New("no private key provided for client mutual TLS config")
	errNoCAForClientTLSConfig     = errors.New("no root certificate provided for client (mutual) TLS config")
)

func SetupTLSConfig(cfg TLSConfig) (tlsConfig *tls.Config, err error) {

	// 单向 TLS 认证
	// 服务端需要设置: 服务端证书, 服务端私钥, 服务端名字
	// 客户端需要设置: 根证书

	// 双向 TLS 认证
	// 服务端需要设置: 服务端证书, 服务端私钥, 服务端名字, 根证书
	// 客户端需要设置: 客户端证书, 客户端私钥, 根证书

	tlsConfig = &tls.Config{}

	if cfg.IsServerConfig {
		// 服务端 TLS 配置

		// 不论是否开启双向 TLS 认证
		// 服务端都需要提供自己的证书和私钥
		if cfg.CertFile == "" {
			return nil, errNoCertForServerTLSConfig
		}
		if cfg.KeyFile == "" {
			return nil, errNoPKeyForServerTLSConfig
		}

		tlsConfig.Certificates = make([]tls.Certificate, 1)
		tlsConfig.Certificates[0], err = tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, err
		}

		// 不论是否开启双向 TLS 认证都需要设置服务端名字
		tlsConfig.ServerName = cfg.ServerName

		// 如果启用了双向 TLS 认证还需要提供根证书来认证客户端证书
		if cfg.EnableMutualTLS {
			if cfg.CAFile == "" {
				return nil, errNoCAForServerMuTLSConfig
			}
			if ca, err := loadCA(cfg.CAFile); err != nil {
				return nil, err
			} else {
				tlsConfig.ClientCAs = ca
				tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
			}
		}

		return tlsConfig, nil
	} else {
		// 服务端 TLS 配置

		// 不论是否开启双向 TLS 认证
		// 客户端都需要提供根证书
		if cfg.CAFile == "" {
			return nil, errNoCAForClientTLSConfig
		}
		if ca, err := loadCA(cfg.CAFile); err != nil {
			return nil, err
		} else {
			tlsConfig.RootCAs = ca
		}

		// 如果开启了双向 TLS 认证
		// 客户端还需要提供自己的证书和私钥
		if cfg.EnableMutualTLS {
			if cfg.CertFile == "" {
				return nil, errNoCertForClientMuTLSConfig
			}
			if cfg.KeyFile == "" {
				return nil, errNoPKeyForClientMuTLSConfig
			}
			tlsConfig.Certificates = make([]tls.Certificate, 1)
			tlsConfig.Certificates[0], err = tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
			if err != nil {
				return nil, err
			}
		}

		return tlsConfig, nil
	}
}

func loadCA(CAFile string) (*x509.CertPool, error) {
	b, err := os.ReadFile(CAFile)
	if err != nil {
		return nil, err
	}
	ca := x509.NewCertPool()
	if ok := ca.AppendCertsFromPEM(b); !ok {
		return nil, fmt.Errorf("failed to parse root certificate: %q", CAFile)
	}
	return ca, nil
}
