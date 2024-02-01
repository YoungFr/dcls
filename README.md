# DCLS (Distributed Commit Log Service)

[toc]

构建一个分布式的提交日志服务。

很多分布式服务的学习资源都是具体的代码，而另外一些则陷于抽象的理论。希望通过本项目的学习，能在理论与实践之间取得一种平衡。分布式系统整体的架构知识参考周志明的《凤凰架构》，其他具体的技术则参考各种书籍和在线文档。

# Part 0 - Overview

TODO

# Part 1 - Service

## 简单的日志服务

在客户端与我们提供的服务之间，请求和响应数据用 JSON 格式来表示，并通过 HTTP 传输。接下来会构建一个简单的提交日志服务，现在我们只需要知道提交日志就是一系列按时间排序且只支持追加写入的记录。

我们在 `./internal/server/log.go` 中定义了一个 `Log` 结构体来表示日志，它是 `Record` 结构体的切片，并受到互斥锁的保护。而一个 `Record` 结构体表示一条具体的记录，记录的内容可以是任意类型，成员 `Offset` 表示的是该条记录在 `Log.records` 中的下标。日志结构体的 `Append` 方法用于追加一条记录，而 `Read` 方法则用于根据下标读取某条记录。

接下来构建 JSON/HTTP 服务。具体来说，我们需要为每个 API 编写一个 `func(http.ResponseWriter, *http.Request)` 类型的处理函数。处理函数中通常包含以下 3 步：

1. 将请求反序列化为 Go 语言对象
2. 处理请求获得结果
3. 将序列化后的结果作为响应

我们在 `./internal/server/http.go` 中定义了 `handleWrite` 和 `handleRead` 两个函数，分别用来处理记录的写入和读取；然后使用 `gorilla/mux` 库为不同的方法和路径注册了对应的处理函数；最后在 `./cmd/server/main.go` 中调用了服务器的 `ListenAndServe` 方法。

## HTTP

详见 [HTTP](./mds/HTTP.md) 中的内容。

## JSON 以及 Go 语言的 `encoding/json` 包

（以下内容来自 [MDN](https://developer.mozilla.org/en-US/docs/Learn/JavaScript/Objects/JSON) 文档）

JSON 的全称是 JavaScript Object Notation（JavaScript 对象表示法），用于以文本的形式表示结构化数据。它最常见的用途是在 Web 应用中表示和传输数据。JSON 的 ABNF 文法表示如下（在 [RFC 8259](https://datatracker.ietf.org/doc/html/rfc8259) 中描述）：

```
// JSON 字符串
JSON-text       = ws value ws

// 空白符
ws              = *(
                  %x20 /               ; Space
                  %x09 /               ; Horizontal tab
                  %x0A /               ; Line feed or New line
                  %x0D )               ; Carriage return

// 值
value           = false / null / true / object / array / number / string
false           = %x66.61.6c.73.65     ; false
null            = %x6e.75.6c.6c        ; null
true            = %x74.72.75.65        ; true

// 对象
object          = begin-object [ member *( value-separator member ) ] end-object
begin-object    = ws %x7B ws           ; { left curly bracket
member          = string name-separator value
name-separator  = ws %x3A ws           ; : colon
value-separator = ws %x2C ws           ; , comma
end-object      = ws %x7D ws           ; } right curly bracket

// 数组
array           = begin-array [ value *( value-separator value ) ] end-array
begin-array     = ws %x5B ws           ; [ left square bracket
end-array       = ws %x5D ws           ; ] right square bracket

// 数字
number          = [ minus ] int [ frac ] [ exp ]
minus           = %x2D                 ; -
int             = zero / ( digit1-9 *DIGIT )
zero            = %x30                 ; 0
digit1-9        = %x31-39              ; 1-9
frac            = decimal-point 1*DIGIT
decimal-point   = %x2E                 ; .
exp             = e [ minus / plus ] 1*DIGIT
e               = %x65 / %x45          ; e E
plus            = %x2B                 ; +    

// 字符串
string          = quotation-mark *char quotation-mark
quotation-mark  = %x22                 ; "
char            = unescaped /
                  escape (
                      %x22 /           ; "    quotation mark  U+0022
                      %x5C /           ; \    reverse solidus U+005C
                      %x2F /           ; /    solidus         U+002F
                      %x62 /           ; b    backspace       U+0008
                      %x66 /           ; f    form feed       U+000C
                      %x6E /           ; n    line feed       U+000A
                      %x72 /           ; r    carriage return U+000D
                      %x74 /           ; t    tab             U+0009
                      %x75 4HEXDIG )   ; uXXXX                U+XXXX
unescaped       = %x20-21 / %x23-5B / %x5D-10FFFF
escape          = %x5C                 ; \
```

当 JSON 以字符串形式存在时，可以用于在网络中传输数据。如果我们想要访问其中的数据，就需要把它转换成一个对象。JavaScript 提供了一个全局的 `JSON` 对象，它有两个静态方法 `parse` 和 `stringify` 来做这种转换，就像下面这样：

```javascript
const json = '{"result":true, "count":42}'
const obj  = JSON.parse(json)
console.log(obj.count)  // 42
console.log(obj.result) // true

// '[3,"false",false]'
console.log(JSON.stringify([new Number(3), new String('false'), new Boolean(false)]))
```

虽然 JSON 使用基于 JavaScript 的语法来描述数据对象，但它仍是一种独立于平台和语言的数据表示和交换格式。比如，Go 语言的 [`encoding/json`](https://pkg.go.dev/encoding/json) 包就提供了将 Go 语言对象序列化为 JSON 字符串和将 JSON 字符串反序列化为 Go 语言对象的方法，其中的核心是 [`Marshal`](https://pkg.go.dev/encoding/json#Marshal) 和 [`Unmarshal`](https://pkg.go.dev/encoding/json#Unmarshal) 函数。这两个函数的文档详细描述了 Go 语言的值和 JSON 的值的对应关系。 一个需要特别注意的地方是 Go 会将 `[]byte` 类型的值序列化为一个使用 `base64` 编码（在 [RFC 4648](https://datatracker.ietf.org/doc/html/rfc4648) 中描述）的字符串。

## 测试

现在来测试下我们的服务。使用 curl 命令发送 POST 请求添加一条记录，然后再发送 GET 请求来查询：

```bash
# 字符串 "TGV0J3MgR28gIzEK" 是 "My First Commit" 的 base64 编码表示
# 正如上一部分最后解释的那样
# 要想让一个字符串能够被反序列化为 []byte 类型的值
# 我们必须提供一个合法的符合 base64 编码规则的字符串
$ curl -X POST localhost:8080 -d '{"record": {"value": "TGV0J3MgR28gIzEK"}}'
{"offset":0}
$ curl -X GET localhost:8080 -d '{"offset": 0}'
{"record":{"value":"TGV0J3MgR28gIzEK","offset":0}}
```

curl 命令行工具用于在客户端和服务器之间传输数据。它的完整描述见 [这里](https://man7.org/linux/man-pages/man1/curl.1.html) 。一些最常见的用法见 [Curl Cookbook](https://catonmat.net/cookbooks/curl) 。

## 使用 Protocol Buffers 数据交换格式

JSON 适用于服务器不需要控制客户端和构建公共 API 时的场景，它的另一个优点在于它是人类可读的形式。但是在构建内部 API 或者需要控制客户端时，我们可以使用其他的数据交换格式，以做到更快的响应、更多的特性和更少的 bug 。本部分对 protobuf 的使用做简要的介绍，详细内容会在 Part 2 中使用 gRPC 时描述。

根据 [官网](https://protobuf.dev/) 的介绍，protobuf 是一种语言和平台无关的用来序列化结构化数据的数据格式。与 XML 和 JSON 相比，它拥有更多的优点，包括 Consistent schemas、Versioning for free、Less boilerplate、Extensibility、Language agnosticism 和 High performance 。官网的 [Overview](https://protobuf.dev/overview/) 页提供了更加详细的介绍。

使用 protobuf 的第一步是安装 protobuf 编译器，它用来编译 `.proto` 文件。最简单的安装方法是在 [GitHub Release](https://github.com/protocolbuffers/protobuf/releases) 页下载合适的版本进行安装，比如在 Linux 系统下可以使用下面的几条命令来完成安装：

```bash
$ wget https://github.com/protocolbuffers/protobuf/releases/download/v25.1/protoc-25.1-linux-x86_64.zip
$ rm -rf /usr/local/protobuf && unzip protoc-25.1-linux-x86_64.zip -d /usr/local/protobuf
$ echo 'export PATH=$PATH:/usr/local/protobuf/bin' >> $HOME/.profile
$ source $HOME/.profile
# 然后输入 protoc --version 查看编译器版本信息
# 没有错误即安装成功
$ protoc --version
libprotoc 25.1
```

接下来就可以编写 `.proto` 文件将上边的 `Record` 类型转换成对应的 protobuf 消息。按照惯例，我们将 protobuf 文件放在 `api` 目录下。在 `./api/v1` 目录下新建 `log.proto` 文件，写入如下内容：

```protobuf
syntax = "proto3";

package log.v1;
option go_package = "github.com/youngfr/api/log_v1";

message Record {
    bytes value = 1;
    uint64 offset = 2;
}
```

关于 protobuf 语法的详细解释见 [proto3 Language Guide](https://protobuf.dev/programming-guides/proto3/) 。在编写好 `.proto` 文件后，为了将其编译为特定的语言，还需要安装对应语言的运行时。对于 Go 语言，可以使用下面这条命令来安装：

```
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

这里要注意，编译器插件 protoc-gen-go 会安装到 `GOBIN` 变量对应的路径下，该路径必须被添加到 `PATH` 环境变量：

```bash
# 查看 GOBIN 的值
$ go env GOBIN
# 如果 GOBIN 没有设置
# 将其设置为 GOPATH/bin 路径
(optional) $ go env -w GOBIN=GOPATH/bin
# 将 GOBIN 添加到 PATH 环境变量
$ echo 'export PATH=$PATH:/home/myl/go/bin' >> $HOME/.profile
$ source $HOME/.profile
```

最后在项目的根目录下新建一个 Makefile 并在其中输入以下内容：

```makefile
compile:
	protoc api/v1/*.proto \
		--go_out=. \
		--go_opt=paths=source_relative \
		--proto_path=.

test:
	go test -race ./...
```

编译参数详见 [Go Generated Code Guide - Compiler Invocation](https://protobuf.dev/reference/go/go-generated/#invocation) 中的解释。此时运行 `make` 命令就可以在 `./api/v1` 路径下看到生成的 `log.pb.go` 文件。

## 编写一个日志包

本节编写一个日志包作为后续服务的基础，一些术语的定义如下：

> Record - the data stored in our log.
>
> Store - the file we store records in.
>
> Index - the file we store index entries in.
>
> Segment - the abstraction that ties a store and an index together.
>
> Log - the abstraction that ties all the segments together.

一个 `store` 结构体用来存放一条一条的记录，详见 `./internal/log/store.go` 中的注释。

一个 `index` 结构体用来保存索引，它的实现使用了内存映射文件，详见 [mmap](./mds/mmap.md) 中的介绍。

一个 `segment` 结构体用来同时保存记录和索引，详见 `./internal/log/segment.go` 中的注释。

一个 `Log` 结构体包括一系列 `segment` 对象，用来处理日志记录的读写。详见 `./internal/log/log.go` 中的注释。

# Part 2 - Network

## RPC 与 REST

RPC 的来龙去脉 —— 来自《凤凰架构》的第 2 章。

1. RPC 的通信成本：远程服务将计算机的工作范围从单机扩展至网络，从本地延伸到远程，是构建分布式系统的首要基础。在 RPC 刚开始出现时，其目的是让计算机能够像调用本地方法一样调用远程方法。在调用本地方法时，计算机（调用者）通常要做传递方法参数、执行被调方法、返回执行结果三件事。但是当调用者与被调者分属不同进程时，如何传递参数和返回结果就成了一个障碍，而 RPC 最初出现时就是被视作一种进程间通信方法来解决这个问题。进程间通信的方法有 **管道** （在相关进程间传递少量字节）、**FIFO** （命名管道，用于在无关进程间传递少量字节）、 **信号** （通知目标进程有某种事件发生）、 **信号量** （进程同步）、 **消息队列** （在进程间传递较多的数据）、 **共享内存** （与信号量结合使用）和 **套接字** （Unix Domain and Internet Domain Socket）。特别要注意的是 Internet Domain Socket 方法，因为它是所有操作系统都提供的标准接口，所以可以用它来进行参数和结果的通信。从而，远程方法的调用细节被隐藏在操作系统底层，做到了“透明的” RPC 调用。但是，这种透明的 RPC 调用却带来了一种通信无成本的假象因而招致了滥用，导致降低了系统性能。随着 1997 年 [Fallacies of Distributed Computing](https://en.wikipedia.org/wiki/Fallacies_of_distributed_computing) 的发表，“**RPC 应该是一种高层次或说语言层次的特征而不是像 IPC 那样是低层次或说系统层次的特征**”的观点成为业界和学界的主流。
2. RPC 的三个基本问题：惠普和 Apollo 提出的 DCE/RPC 和 Sun 公司提出的 ONC RPC 是如今各种 RPC 协议和框架的鼻祖，从它们开始，各种 RPC 协议和框架无非要解决 **如何表示数据** （使用中立的数据流格式进行序列化和反序列化）、 **如何传递数据** （一般基于 TCP 和 UDP 等传输层协议）和 **如何表示方法** （接口描述语言 IDL 和 UUID 的使用）三个问题。
3. RPC 的统一与分裂：早期的 DCE/RPC 、 ONC RPC 、 DCOM 和 CORBA 都因为各种问题从未大规模流行过，已经被扫进了计算机历史博物馆。最终，于 1998 年诞生的数据交换格式 XML 和于 1999 年诞生的 Web Service 远程服务协议取得了统一，风头一时无两。但是，“贪婪的” Web Service 试图通过制定一整套协议来解决分布式计算中的事务、一致性、事件、通知、业务描述、安全等各种功能，这些数不清的协议极大地增加了人们的学习负担，大家对 Web Service 的热情很快冷却。人们逐渐意识到：很难有一个同时满足简单、普适、高性能的完美的 RPC 协议。于是 RPC 协议和框架又一次走向分裂。现在的 RPC 框架往往都针对某个特点作为其主要发展方向，比如 **面向对象** （RMI 和 .NET Remoting）、 **性能** （gRPC 和 Thrift）和 **简化** （JSON-RPC）。最近的 RPC 框架则普遍聚焦于提供负载均衡、服务注册、可观测性等更高层次的能力的支持，而通过插件化的形式来解决上述的三个基本问题。比如，用户可以自己选择要使用的数据交换格式和数据传输协议。

REST：TODO

## 使用 gRPC 框架

gRPC 的基础知识 —— 来自官网的 [Introduction to gRPC](https://grpc.io/docs/what-is-grpc/introduction/) 、[Core concepts, Architecture and Lifecycle](https://grpc.io/docs/what-is-grpc/core-concepts/) 和 [FAQ](https://grpc.io/docs/what-is-grpc/faq/) 页面。

在 gRPC 中，客户端可以直接调用位于不同机器上的服务端方法。像很多 RPC 系统一样，gRPC 也需要首先定义服务、声明可以远程调用的方法及其参数和返回值，然后服务端需要实现这个接口并运行一个 gRPC 服务器来处理客户端调用请求；在服务端则有一个 stub 来提供和服务端相同的方法。

gRPC 使用 Protocol Buffers 作为其接口描述语言（IDL）和数据交换格式（也可以使用其他格式，比如 JSON）。在定义服务端方法时，gRPC 允许我们定义 4 种不同的类型：

- Unary RPC：客户端的请求和服务端的响应都是单条消息，就像普通的函数那样。

  ```protobuf
  rpc SayHello(HelloRequest) returns (HelloResponse);
  ```

- Server streaming RPC：客户端请求是单条消息，服务端响应则是一个消息流。客户端可以不断地从中读取消息直到没有更多的消息。

  ```protobuf
  rpc LotsOfReplies(HelloRequest) returns (stream HelloResponse);
  ```

- Client streaming RPC：客户端以流的形式将一系列消息写入并发送，然后等待服务端读取流并回复响应。

  ```protobuf
  rpc LotsOfGreetings(stream HelloRequest) returns (HelloResponse);
  ```

- Bidirectional streaming RPC：客户端和服务端都使用一个读写流来发送消息，并且这两个流的操作是彼此独立的，从而客户端和服务端可以以任何顺序进行读和写。

  ```protobuf
  rpc BidiHello(stream HelloRequest) returns (stream HelloResponse);
  ```

在 Go 语言中使用 gRPC 需要安装 [grpc-go](https://github.com/grpc/grpc-go) 库，基本示例、文档和各种特性的示例也在这个仓库中。 [helloworld](https://github.com/grpc/grpc-go/tree/master/examples/helloworld) 是一个最基础的例子， [route_guide](https://github.com/grpc/grpc-go/tree/master/examples/route_guide) 则是一个更复杂的例子，官网的 [Basic Tutorial](https://grpc.io/docs/languages/go/basics/) 页面介绍的就是这个示例。gRPC 提供的其他各种特性在 [Documentation](https://github.com/grpc/grpc-go/tree/master/Documentation) 中描述，使用示例则存放在 [features](https://github.com/grpc/grpc-go/tree/master/examples/features) 目录下。

接下来开始实现我们的 RPC 服务。

首先在 `log.proto` 中新增服务的定义，然后运行下面两条命令来安装 gRPC 插件以编译 gRPC 服务：

```bash
$ go get google.golang.org/grpc
$ go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
```

修改 Makefile 后运行 `make compile` 命令就会发现在 `api/v1` 目录下生成了 `log_grpc.pb.go` 文件，里边有一个实现好的客户端和一系列需要我们实现的服务端 API ，接下来的任务就是分别实现它们。详见代码中的注释。

## 认证与授权

实现安全的步骤：

- 数据加密 —— 使用 [TLS](https://www.cloudflare.com/zh-cn/learning/ssl/transport-layer-security-tls/) 防止 [中间人（Man-In-The-Middle attack, MITM）攻击](https://en.wikipedia.org/wiki/Man-in-the-middle_attack) 。我们接下来会为服务器和客户端获取证书并告诉 gRPC 使用这些证书来进行 TLS 加密通信。
- 认证 —— 在公开的 Web 服务中使用的是单向 TLS 认证。即只需要服务端提供证书，客户端通过证书来验证服务端身份，而服务端对客户端的认证则通过用户名加密码和 Token 的方式来完成。 [双向 TLS 认证](https://www.cloudflare.com/zh-cn/learning/access-management/what-is-mutual-tls/) 则用于私密服务，比如分布式系统中机器之间的通信。在双向 TLS 认证中，客户端和服务端都需要提供证书来验证对方身份。
- 授权 —— 当一项资源可以被多个用户访问时，系统需要控制一个用户可以看到哪些数据、进行哪些操作。



## 系统的可观测性

TODO

# Part 3 - Distribute

TODO

# Part 4 - Deploy

TODO
