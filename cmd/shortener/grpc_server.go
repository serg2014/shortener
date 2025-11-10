package main

import (
	"context"

	pb "github.com/serg2014/shortener/cmd/shortener/proto"
	"github.com/serg2014/shortener/internal/app"
	"github.com/serg2014/shortener/internal/config"
	"github.com/serg2014/shortener/internal/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type GrpcServer struct {
	pb.UnimplementedShortenerServiceServer

	app *app.MyApp
}

func trustedInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if info.FullMethod == "/shortener.ShortenerService/InternalStats" {
		trust := false
		// выполняем действия перед вызовом метода
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			values := md.Get("X-Real-IP")
			if len(values) > 0 {
				// ключ содержит слайс строк, получаем первую строку
				ip := values[0]
				trust = config.Config.TrustedSubnet.IsTrusted(ip)
			}
		}
		if !trust {
			code := codes.PermissionDenied
			return nil, status.Error(code, code.String())

		}
	}
	// Возвращать ответ и ошибку от фактического обработчика
	return handler(ctx, req)
}

func (s *GrpcServer) InternalStats(ctx context.Context, request *pb.InternalStatsRequest) (*pb.InternalStatsResponse, error) {
	data, err := s.app.InternalStats(ctx)
	if err != nil {
		// TODO нужен логер чтобы сам делала префикс grpc
		logger.Log.Error("grpc: InternalStats", zap.Error(err))
		code := codes.Internal
		return nil, status.Error(codes.Internal, code.String())
	}
	return &pb.InternalStatsResponse{
		Urls:  uint32(data.Urls),
		Users: uint32(data.Users),
	}, nil
}
