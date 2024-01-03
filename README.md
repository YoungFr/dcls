# DCLS (Distributed Commit Log Service)

构建一个分布式的提交日志服务。

很多分布式服务的学习资源都是具体的代码，而另外一些则陷于抽象的理论。本项目来自于 *Distributed Services with Go* 这本书，希望能在理论与实践之间取得一种平衡。

# Part 1 - Service

在客户端与我们提供的服务之间，请求和响应是用 JSON 来表示的，并通过 HTTP 来传输。接下来会构建一个简单的提交日志服务，现在只需要知道提交日志就是一系列按时间排序且只能追加写入的记录。



我们在 `./internal/server/log.go` 中定义了一个 `Log` 结构体来表示日志，它是 `Record` 结构体的切片，并受到互斥锁的保护。而一个 `Record` 结构体表示一条具体的记录，记录的内容可以是任意类型，成员 `Offset` 表示的是该条记录在 `Log.records` 中的下标。日志结构体的 `Append` 方法用于追加某条记录，而 `Read` 方法则用于根据下标读取某条记录。



接下来构建 JSON/HTTP 服务。需要为每一个 API 编写一个 `func(http.ResponseWriter, *http.Request)` 类型的处理函数。函数中通常包含以下 3 步：

1. 将请求反序列化为 Go 结构体
2. 处理请求获得结果
3. 将序列化后的结果作为响应

我们在 `./internal/server/http.go` 中定义了 `handleWrite` 和 `handleRead` 两个函数，分别用来处理记录的写入和读取；然后使用 `gorilla/mux` 库为不同的方法和路径注册对应的处理函数；最后在 `./cmd/server/main.go` 中调用 服务器的 `ListenAndServe` 方法。

# Part 2 - Network

TODO

# Part 3 - Distribute

TODO

# Part 4 - Deploy

TODO
