# 证书存放的路径
CONFIG_PATH=${HOME}/.dcls/

# 初始化证书存放的路径
.PHONY: init
init:
	rm  -rf  ${CONFIG_PATH}
	mkdir -p ${CONFIG_PATH}
	cp cfgfiles/authorize/* ${CONFIG_PATH}

# 根证书
# 服务端证书
# 超级用户 (root user) 的客户端证书
# 普通用户 (ordinary user) 的客户端证书
# 只读用户 (read-only user) 的客户端证书
.PHONY: gencert
gencert:
	cfssl gencert -initca cfgfiles/certs/ca-csr.json | cfssljson -bare ca

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=cfgfiles/certs/ca-config.json \
		-profile=server \
		cfgfiles/certs/server-csr.json | cfssljson -bare server
	
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=cfgfiles/certs/ca-config.json \
		-profile=client \
		-cn="root user" \
		cfgfiles/certs/client-csr.json | cfssljson -bare root-client
	
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=cfgfiles/certs/ca-config.json \
		-profile=client \
		-cn="ordinary user" \
		cfgfiles/certs/client-csr.json | cfssljson -bare ordinary-client
	
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=cfgfiles/certs/ca-config.json \
		-profile=client \
		-cn="readonly user" \
		cfgfiles/certs/client-csr.json | cfssljson -bare readonly-client
	
	mv *.pem *.csr ${CONFIG_PATH}

.PHONY: test
test:
	go test -race ./tests

# 编译所有 protobuf 文件
.PHONY: compile
compile:
	protoc api/v1/*.proto \
		--go_out=. \
		--go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		--proto_path=.
