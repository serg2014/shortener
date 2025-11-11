package main

import (
	"context"
	"errors"
	"time"

	pb "github.com/serg2014/shortener/cmd/shortener/proto"
	"github.com/serg2014/shortener/internal/app"
	"github.com/serg2014/shortener/internal/auth"
	"github.com/serg2014/shortener/internal/logger"
	"github.com/serg2014/shortener/internal/models"
	"github.com/serg2014/shortener/internal/storage"
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

func metadataInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	resp, err := handler(ctx, req)

	md, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		grpc.SetTrailer(ctx, md)
	}
	return resp, err
}

func (s *GrpcServer) InternalStats(ctx context.Context, request *pb.InternalStatsRequest) (*pb.InternalStatsResponse, error) {
	data, err := s.app.InternalStats(ctx)
	if err != nil {
		// TODO нужен логер чтобы сам делала префикс grpc
		logger.Log.Error("InternalStats", zap.Error(err))
		code := codes.Internal
		return nil, status.Error(code, code.String())
	}
	return &pb.InternalStatsResponse{
		Urls:  uint32(data.Urls),
		Users: uint32(data.Users),
	}, nil
}

func (s *GrpcServer) Ping(ctx context.Context, request *pb.PingRequest) (*pb.PingResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if err := s.app.Ping(ctx); err != nil {
		logger.Log.Error("ping", zap.Error(err))
		code := codes.Internal
		return nil, status.Error(code, code.String())
	}
	return &pb.PingResponse{}, nil
}

func (s *GrpcServer) DeleteUserURLS(ctx context.Context, request *pb.DeleteUserURLSRequest) (*pb.DeleteUserURLSResponse, error) {
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		logger.Log.Error("can not find userid", zap.Error(err))
		code := codes.Internal
		return nil, status.Error(code, code.String())
	}
	req := models.RequestForDeleteURLS(request.Shorts)
	err = s.app.DeleteUserURLS(ctx, req, userID)
	if err != nil {
		logger.Log.Error("can not find userid", zap.Error(err))
		code := codes.Internal
		return nil, status.Error(code, code.String())
	}

	return &pb.DeleteUserURLSResponse{}, nil
}

func (s *GrpcServer) GetURL(ctx context.Context, request *pb.GetURLRequest) (*pb.GetURLResponse, error) {
	origURL, ok, err := s.app.Get(ctx, request.Short)
	var code codes.Code
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrDeleted):
			// TODO http.StatusGone
			code = codes.Internal
		default:
			code = codes.Internal
		}
		logger.Log.Error("error in a.Get", zap.Error(err))
		return nil, status.Error(code, code.String())
	}
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "bad short")
	}

	return &pb.GetURLResponse{Url: origURL}, nil
}

func (s *GrpcServer) ShortURLS(ctx context.Context, request *pb.ShortURLSRequest) (*pb.ShortURLSResponse, error) {
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		logger.Log.Error("can not find userid", zap.Error(err))
		code := codes.Internal
		return nil, status.Error(code, code.String())
	}

	// TODO проверить что прислали урл. correlation_id должен быть уникальным
	req := models.RequestBatch{}
	for _, pbItem := range request.Items {
		// TODO выделить памяти столько, сколько надо
		req = append(req, models.RequestBatchItem{CorrelationID: pbItem.CorrelationId, OriginalURL: pbItem.OriginalUrl})
	}

	resp, err := s.app.GenerateShortURLBatch(ctx, req, userID)
	if err != nil {
		logger.Log.Error(
			"can not generate short batch",
			zap.Error(err),
		)
		code := codes.Internal
		return nil, status.Error(code, code.String())
	}

	response := pb.ShortURLSResponse{
		Items: make([]*pb.ShortURLSResponse_Item, len(resp)),
	}
	for i := range resp {
		response.Items[i] = &pb.ShortURLSResponse_Item{
			CorrelationId: resp[i].CorrelationID,
			ShortUrl:      resp[i].ShortURL,
		}
	}
	return &response, nil
}

func (s *GrpcServer) GetUserURLS(ctx context.Context, request *pb.GetUserURLSRequest) (*pb.GetUserURLSResponse, error) {
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		logger.Log.Error("can not find userid", zap.Error(err))
		code := codes.Internal
		return nil, status.Error(code, code.String())
	}
	data, err := s.app.GetUserURLS(ctx, userID)
	if err != nil {
		logger.Log.Error("GetUserURLS", zap.Error(err))
		code := codes.Internal
		return nil, status.Error(code, code.String())
	}
	if len(data) == 0 {
		// code := http.StatusNoContent
		// http.Error(w, http.StatusText(code), code)
		// TODO
		code := codes.Internal
		return nil, status.Error(code, code.String())
	}

	response := pb.GetUserURLSResponse{
		Items: make([]*pb.GetUserURLSResponse_Item, len(data)),
	}
	for i := range data {
		response.Items[i] = &pb.GetUserURLSResponse_Item{
			OriginalUrl: data[i].OriginalURL,
			ShortUrl:    data[i].ShortURL,
		}
	}
	return &response, nil
}
