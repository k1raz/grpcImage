package limiter

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ConcurrencyLimiter struct {
	uploadDownloadSem chan struct{}
	listSem           chan struct{}
}

func NewConcurrencyLimiter(uploadDownloadLimit, listLimit int) *ConcurrencyLimiter {
	return &ConcurrencyLimiter{
		uploadDownloadSem: make(chan struct{}, uploadDownloadLimit),
		listSem:           make(chan struct{}, listLimit),
	}
}

func (l *ConcurrencyLimiter) LimitUploadDownload(ctx context.Context) error {
	select {
	case l.uploadDownloadSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return status.Error(codes.ResourceExhausted, "upload/download limit exceeded")
	}
}

func (l *ConcurrencyLimiter) ReleaseUploadDownload() {
	<-l.uploadDownloadSem
}

func (l *ConcurrencyLimiter) LimitList(ctx context.Context) error {
	select {
	case l.listSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return status.Error(codes.ResourceExhausted, "list limit exceeded")
	}
}

func (l *ConcurrencyLimiter) ReleaseList() {
	<-l.listSem
}

func (l *ConcurrencyLimiter) UnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	var err error

	if info.FullMethod == "/api.FileService/ListFiles" {
		if err = l.LimitList(ctx); err != nil {
			return nil, err
		}
		defer l.ReleaseList()
	}

	return handler(ctx, req)
}

func (l *ConcurrencyLimiter) StreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	var err error

	if info.FullMethod == "/api.FileService/UploadFile" || info.FullMethod == "/api.FileService/DownloadFile" {
		if err = l.LimitUploadDownload(ss.Context()); err != nil {
			return err
		}
		defer l.ReleaseUploadDownload()
	}

	return handler(srv, ss)
}
