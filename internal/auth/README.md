# 认证与授权

# 认证

TODO

# 授权

授权是控制某个用户能看到哪些资源、进行哪些操作的过程。

最简单的授权实现方式是 [访问控制列表（Access Control List, ACL）](https://en.wikipedia.org/wiki/Access-control_list) 。它是一个每行的形式都类似于 `A, B, C` 的规则表，每一行的含义是 " **Subject** A is permitted to do **Action** B on **Object** C " ，即 “主体 A 可以对客体 B 执行 C 操作”。我们基于 [casbin](https://github.com/casbin/casbin) 库来实现 ACL 授权。

术语

authorization enforcement —— 授权执行

policy management —— 策略管理

安装 casbin 库：

```bash
$ go get github.com/casbin/casbin/v2
```

