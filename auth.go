package service

import (
	"context"

	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/dstore"
	"github.com/keys-pub/keys/encoding"
	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	status "google.golang.org/grpc/status"
)

func generateToken() string {
	return encoding.MustEncode(keys.Rand32()[:], encoding.Base62)
}

type authInterceptor struct {
	tokens    map[string]string
	allowlist *dstore.StringSet
}

func newAuthInterceptor() *authInterceptor {
	// We don't need auth for the following methods.
	allowlist := dstore.NewStringSet(
		"/service.RPC/AccountCreate",
		"/service.RPC/AuthUnlock",
		"/service.RPC/AuthLock",
		"/service.RPC/AuthStatus",
		"/service.RPC/Rand",
		"/service.RPC/RandPassword",
	)

	return &authInterceptor{
		tokens:    map[string]string{},
		allowlist: allowlist,
	}
}

func (a *authInterceptor) streamInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := a.authorize(stream.Context(), info.FullMethod); err != nil {
		return err
	}
	return handler(srv, stream)
}

func (a *authInterceptor) unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if err := a.authorize(ctx, info.FullMethod); err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

func (a *authInterceptor) checkToken(token string) error {
	for _, t := range a.tokens {
		if t == token {
			return nil
		}
	}
	logger.Infof("Invalid auth token")
	return status.Error(codes.Unauthenticated, "invalid token")
}

func (a *authInterceptor) registerToken(client string) string {
	token := generateToken()
	logger.Debugf("Auth register client (%q)", client)
	a.tokens[client] = token
	return token
}

func (a *authInterceptor) clearTokens() {
	a.tokens = map[string]string{}
}

func (a *authInterceptor) authorize(ctx context.Context, method string) error {
	// No authorization needed for allowed methods.
	if a.allowlist.Contains(method) {
		// TODO: Auth token could be set and invalid here
		logger.Infof("Authorization is not required for %s", method)
		return nil
	}

	logger.Infof("Authorize %s", method)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if len(md["authorization"]) == 0 {
			logger.Warningf("Auth token missing from request")
			return status.Error(codes.Unauthenticated, "authorization missing")
		}
		token := md["authorization"][0]
		return a.checkToken(token)
	}
	return status.Error(codes.Unauthenticated, "no authorization in context")
}
