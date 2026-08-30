// Package grpc provides gRPC server implementation for the URL shortener service.
//
// The package exposes three RPC methods that reuse the same storage layer as the
// HTTP handlers:
//   - ShortenURL — shorten a URL (POST /api/shorten equivalent)
//   - ExpandURL — expand a short URL to original (GET /{id} equivalent)
//   - ListUserURLs — list all URLs for the authenticated user
//
// Authentication is performed via an interceptor that reads the "authorization"
// metadata from incoming requests and extracts the user ID.
package grpc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"

	auth "github.com/andrea20024/go-musthave-shortener-tpl/internal/auth"
	storage "github.com/andrea20024/go-musthave-shortener-tpl/internal/repository"
	pb "github.com/andrea20024/go-musthave-shortener-tpl/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const authorizationMetadata = "authorization"

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

// userID is the context key for the user ID.
const userID contextKey = "userID"

// Server implements the gRPC ShortenerService.
type Server struct {
	pb.UnimplementedShortenerServiceServer
	repo    storage.Repository
	baseURL string
}

// NewServer creates a new gRPC server with the given repository and base URL.
func NewServer(repo storage.Repository, baseURL string) *Server {
	return &Server{repo: repo, baseURL: baseURL}
}

// StartGRPCServer creates and starts the gRPC server on the given address.
// The server runs in a background goroutine; call GracefulStop() to shut it down.
func StartGRPCServer(addr string, repo storage.Repository, baseURL string) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s := grpc.NewServer(grpc.UnaryInterceptor(authInterceptor))
	pb.RegisterShortenerServiceServer(s, NewServer(repo, baseURL))

	go func() {
		if err := s.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	return s, nil
}

// authInterceptor extracts the user ID from the "authorization" metadata
// and injects it into the request context.
func authInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	authValues := md[authorizationMetadata]
	if len(authValues) == 0 || authValues[0] == "" {
		return nil, status.Error(codes.Unauthenticated, "authorization header required")
	}

	// Extract userID from "Bearer <token>" or raw token format.
	token := strings.TrimPrefix(authValues[0], "Bearer ")

	userIDFromToken, valid := auth.VerifyToken(token)
	if !valid {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	ctx = context.WithValue(ctx, userID, userIDFromToken)
	return handler(ctx, req)
}

// getUserIDFromContext extracts the user ID from the gRPC context.
func getUserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(userID).(string)
	if !ok || userID == "" {
		return "", errors.New("user ID not found in context")
	}
	return userID, nil
}

// GenerateShortURL generates a cryptographically secure random short URL key
// using 6 bytes from crypto/rand, encoded as URL-safe base64.
func GenerateShortURL() (string, error) {
	b := make([]byte, 6)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:8], nil
}

// ShortenURL handles URL shortening via gRPC.
func (s *Server) ShortenURL(ctx context.Context, req *pb.URLShortenRequest) (*pb.URLShortenResponse, error) {
	if req.Url == "" {
		return nil, status.Error(codes.InvalidArgument, "url is required")
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	shortURL, err := GenerateShortURL()
	if err != nil {
		return nil, status.Error(codes.Internal, "generate short URL failed")
	}

	err = s.repo.Add(shortURL, req.Url, userID)
	if err != nil {
		return nil, status.Error(codes.AlreadyExists, "url already exists")
	}

	result := fmt.Sprintf("%s/%s", s.baseURL, shortURL)

	return &pb.URLShortenResponse{Result: result}, nil
}

// ExpandURL handles URL expansion via gRPC.
func (s *Server) ExpandURL(ctx context.Context, req *pb.URLExpandRequest) (*pb.URLExpandResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	originalURL, err := s.repo.Get(req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, "url not found")
	}

	return &pb.URLExpandResponse{Result: originalURL}, nil
}

// ListUserURLs handles listing user URLs via gRPC.
func (s *Server) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*pb.UserURLsResponse, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	userURLs, err := s.repo.GetUserURLs(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	urls := make([]*pb.URLData, 0, len(userURLs))
	for _, u := range userURLs {
		fullURL := fmt.Sprintf("%s/%s", s.baseURL, u.ShortURL)
		urls = append(urls, &pb.URLData{
			ShortUrl:    fullURL,
			OriginalUrl: u.OriginalURL,
		})
	}

	return &pb.UserURLsResponse{Url: urls}, nil
}
