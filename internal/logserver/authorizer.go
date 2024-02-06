package logserver

import (
	"context"

	"github.com/youngfr/dcls/internal/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// 访问控制接口
// 无论是否使用 casbin 库来做访问控制
// 都需要实现 Authorize 方法
type Authorizer interface {
	Authorize(subject, object, action string) error
}

// 在 auth 包中的 *auth.Authorizer 实现了 Authorizer 接口
var _ Authorizer = (*auth.Authorizer)(nil)

// actions
const (
	objects      = "all logs"
	appendAction = "append"
	readAction   = "read"
	resetAction  = "reset"
)

func authenticate(ctx context.Context) (context.Context, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return ctx, status.New(
			codes.Unknown,
			"couldn't find peer info",
		).Err()
	}

	if peer.AuthInfo == nil {
		return ctx, status.New(
			codes.Unauthenticated,
			"no transport security being used",
		).Err()
	}

	tlsInfo := peer.AuthInfo.(credentials.TLSInfo)
	subject := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
	ctx = context.WithValue(ctx, subjectContextKey{}, subject)

	return ctx, nil
}

func subject(ctx context.Context) string {
	return ctx.Value(subjectContextKey{}).(string)
}

type subjectContextKey struct{}
