# 证书存放的路径
CONFIG_PATH=${HOME}/.dcls/

# 初始化证书存放的路径
.PHONY: init
init:
	rm  -rf  ${CONFIG_PATH}
	mkdir -p ${CONFIG_PATH}

# 根证书, 服务端证书, 客户端证书
.PHONY: gencert
gencert:
	cfssl gencert -initca certscfg/ca-csr.json | cfssljson -bare ca

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=certscfg/ca-config.json \
		-profile=server \
		certscfg/server-csr.json | cfssljson -bare server
	
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=certscfg/ca-config.json \
		-profile=client \
		certscfg/client-csr.json | cfssljson -bare client
	
	mv *.pem *.csr ${CONFIG_PATH}

.PHONY: test
test:
	go test -race ./...

# 编译所有 protobuf 文件
.PHONY: compile
compile:
	protoc api/v1/*.proto \
		--go_out=. \
		--go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		--proto_path=.
