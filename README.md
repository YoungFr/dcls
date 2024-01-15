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

虽然 JSON 使用基于 JavaScript 的语法来描述数据对象，但它仍是一种独立于平台和语言的数据表示和交换格式。比如，Go 语言的 [`encoding/json`](https://pkg.go.dev/encoding/json) 包就提供了将 Go 语言对象序列化为 JSON 字符串和将 JSON 字符串反序列化为 Go 语言对象的方法，其中的核心是 [`Marshal`](https://pkg.go.dev/encoding/json#Marshal) 和 [`Unmarshal`](https://pkg.go.dev/encoding/json#Unmarshal) 函数。这两个函数的文档详细描述了 Go 语言的值和 JSON 的值的对应关系。 一个需要特别注意的地方是 Go 会将 `[]byte` 类型的值编码为一个使用 `base64` 编码（在 [RFC 4648](https://datatracker.ietf.org/doc/html/rfc4648) 中描述）的字符串。

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

TODO

## 使用 gRPC 框架

TODO

# Part 3 - Distribute

TODO

# Part 4 - Deploy

TODO
