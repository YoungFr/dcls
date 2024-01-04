# DCLS (Distributed Commit Log Service)

构建一个分布式的提交日志服务。

很多分布式服务的学习资源都是具体的代码，而另外一些则陷于抽象的理论。本项目来自于 *Distributed Services with Go* 这本书，希望能在理论与实践之间取得一种平衡。

# Part 1 - Service

## 简单的日志服务

在客户端与我们提供的服务之间，请求和响应是用 JSON 来表示的，并通过 HTTP 来传输。接下来会构建一个简单的提交日志服务，现在只需要知道提交日志就是一系列按时间排序且只支持追加写入的记录。



我们在 `./internal/server/log.go` 中定义了一个 `Log` 结构体来表示日志，它是 `Record` 结构体的切片，并受到互斥锁的保护。而一个 `Record` 结构体表示一条具体的记录，记录的内容可以是任意类型，成员 `Offset` 表示的是该条记录在 `Log.records` 中的下标。日志结构体的 `Append` 方法用于追加一条记录，而 `Read` 方法则用于根据下标读取某条记录。



接下来构建 JSON/HTTP 服务。我们需要为每个 API 编写一个 `func(http.ResponseWriter, *http.Request)` 类型的处理函数。处理函数中通常包含以下 3 步：

1. 将请求反序列化为 Go 结构体
2. 处理请求获得结果
3. 将序列化后的结果作为响应

我们在 `./internal/server/http.go` 中定义了 `handleWrite` 和 `handleRead` 两个函数，分别用来处理记录的写入和读取；然后使用 `gorilla/mux` 库为不同的方法和路径注册对应的处理函数；最后在 `./cmd/server/main.go` 中调用服务器的 `ListenAndServe` 方法。

## JSON 以及 Go 语言的 `encoding/json` 包

（以下内容来自 [MDN](https://developer.mozilla.org/en-US/docs/Learn/JavaScript/Objects/JSON) 文档）

JSON 的全称是 JavaScript Object Notation（JavaScript 对象表示法），用于以文本的形式表示结构化数据。它最常见的用途是在 Web 应用中表示和传输数据。JSON 的 ABNF 文法表示如下（来自 [RFC 8259](https://datatracker.ietf.org/doc/html/rfc8259)）：

```
// JSON 字符串
JSON-text = ws value ws

// 空白符
ws              = *(
                  %x20 /              ; Space
                  %x09 /              ; Horizontal tab
                  %x0A /              ; Line feed or New line
                  %x0D )              ; Carriage return

// 值
value           = false / null / true / object / array / number / string
false           = %x66.61.6c.73.65    ; false
null            = %x6e.75.6c.6c       ; null
true            = %x74.72.75.65       ; true

// 对象
object          = begin-object [ member *( value-separator member ) ] end-object
begin-object    = ws %x7B ws          ; { left curly bracket
member          = string name-separator value
name-separator  = ws %x3A ws          ; : colon
value-separator = ws %x2C ws          ; , comma
end-object      = ws %x7D ws          ; } right curly bracket

// 数组
array           = begin-array [ value *( value-separator value ) ] end-array
begin-array     = ws %x5B ws          ; [ left square bracket
end-array       = ws %x5D ws          ; ] right square bracket

// 数字
number         = [ minus ] int [ frac ] [ exp ]
minus          = %x2D                 ; -
int            = zero / ( digit1-9 *DIGIT )
zero           = %x30                 ; 0
digit1-9       = %x31-39              ; 1-9
frac           = decimal-point 1*DIGIT
decimal-point  = %x2E                 ; .
exp            = e [ minus / plus ] 1*DIGIT
e              = %x65 / %x45          ; e E
plus           = %x2B                 ; +    

// 字符串
string         = quotation-mark *char quotation-mark
quotation-mark = %x22                 ; "
char           = unescaped /
                 escape (
                      %x22 /          ; "    quotation mark  U+0022
                      %x5C /          ; \    reverse solidus U+005C
                      %x2F /          ; /    solidus         U+002F
                      %x62 /          ; b    backspace       U+0008
                      %x66 /          ; f    form feed       U+000C
                      %x6E /          ; n    line feed       U+000A
                      %x72 /          ; r    carriage return U+000D
                      %x74 /          ; t    tab             U+0009
                      %x75 4HEXDIG )  ; uXXXX                U+XXXX
unescaped      = %x20-21 / %x23-5B / %x5D-10FFFF
escape         = %x5C                 ; \
```

当 JSON 以字符串形式存在时，可以用于在网络中传输数据。如果我们想要访问其中的数据，就需要把它转换成一个对象。JavaScript 语言提供了一个全局的 `JSON` 对象，它有两个方法来做这种转换。

# Part 2 - Network

TODO

# Part 3 - Distribute

TODO

# Part 4 - Deploy

TODO
