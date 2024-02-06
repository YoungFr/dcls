package auth

import (
	"fmt"

	"github.com/casbin/casbin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Authorizer struct {
	enforcer *casbin.Enforcer
}

// 参数 model 是使用的访问控制模型的 conf 文件的路径
// 参数 policy 是定义了具体的策略的文件的路径
func NewAuthorizer(model, policy string) *Authorizer {
	enforcer := casbin.NewEnforcer(model, policy)
	return &Authorizer{
		enforcer: enforcer,
	}
}

// 如果 subject 可以对 object 执行 action 操作则返回空
func (a *Authorizer) Authorize(subject, object, action string) error {
	if !a.enforcer.Enforce(subject, object, action) {
		msg := fmt.Sprintf("%s is not permitted to %s to %s", subject, action, object)
		return status.New(codes.PermissionDenied, msg).Err()
	}
	return nil
}
