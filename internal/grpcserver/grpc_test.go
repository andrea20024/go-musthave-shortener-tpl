package grpc

import (
	"context"
	"testing"

	auth "github.com/andrea20024/go-musthave-shortener-tpl/internal/auth"
	storage "github.com/andrea20024/go-musthave-shortener-tpl/internal/repository"
	pb "github.com/andrea20024/go-musthave-shortener-tpl/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func init() {
	auth.Init("test-secret-key-for-grpc-tests")
}

// withUserID adds a user ID to the context for testing purposes.
// It uses the same context key as the auth interceptor.
func withUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userID, id)
}

func TestShortenURL(t *testing.T) {
	repo := storage.NewMapRepository()
	srv := NewServer(repo, "http://localhost:8080")

	tests := []struct {
		name       string
		url        string
		userID     string
		wantStatus codes.Code
	}{
		{
			name:       "successful shorten",
			url:        "https://example.com/test",
			userID:     "test-user",
			wantStatus: codes.OK,
		},
		{
			name:       "empty url",
			url:        "",
			userID:     "test-user",
			wantStatus: codes.InvalidArgument,
		},
		{
			name:       "unauthorized",
			url:        "https://example.com/test",
			userID:     "",
			wantStatus: codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.userID != "" {
				ctx = withUserID(ctx, tt.userID)
			}

			resp, err := srv.ShortenURL(ctx, &pb.URLShortenRequest{Url: tt.url})
			st, _ := status.FromError(err)

			assert.Equal(t, tt.wantStatus, st.Code())
			if tt.wantStatus == codes.OK {
				assert.NotEmpty(t, resp.Result)
				assert.Contains(t, resp.Result, "http://localhost:8080")
			}
		})
	}
}

func TestExpandURL(t *testing.T) {
	repo := storage.NewMapRepository()
	srv := NewServer(repo, "http://localhost:8080")

	// Add a URL first
	shortKey, _ := auth.GenerateShortURL()
	repo.Add(shortKey, "https://example.com/target", "user1")

	tests := []struct {
		name       string
		id         string
		userID     string
		wantStatus codes.Code
		wantResult string
	}{
		{
			name:       "successful expand",
			id:         shortKey,
			userID:     "user1",
			wantStatus: codes.OK,
			wantResult: "https://example.com/target",
		},
		{
			name:       "url not found",
			id:         "nonexistent",
			userID:     "user1",
			wantStatus: codes.NotFound,
		},
		{
			name:       "empty id",
			id:         "",
			userID:     "user1",
			wantStatus: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.userID != "" {
				ctx = withUserID(ctx, tt.userID)
			}

			resp, err := srv.ExpandURL(ctx, &pb.URLExpandRequest{Id: tt.id})
			st, _ := status.FromError(err)

			assert.Equal(t, tt.wantStatus, st.Code())
			if tt.wantStatus == codes.OK {
				assert.Equal(t, tt.wantResult, resp.Result)
			}
		})
	}
}

func TestListUserURLs(t *testing.T) {
	repo := storage.NewMapRepository()
	srv := NewServer(repo, "http://localhost:8080")

	// Add some URLs
	key1, _ := auth.GenerateShortURL()
	key2, _ := auth.GenerateShortURL()
	repo.Add(key1, "https://example.com/1", "user1")
	repo.Add(key2, "https://example.com/2", "user1")

	tests := []struct {
		name       string
		userID     string
		wantStatus codes.Code
		wantCount  int
	}{
		{
			name:       "list user URLs",
			userID:     "user1",
			wantStatus: codes.OK,
			wantCount:  2,
		},
		{
			name:       "empty list for unknown user",
			userID:     "unknown-user",
			wantStatus: codes.OK,
			wantCount:  0,
		},
		{
			name:       "unauthorized",
			userID:     "",
			wantStatus: codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.userID != "" {
				ctx = withUserID(ctx, tt.userID)
			}

			resp, err := srv.ListUserURLs(ctx, &emptypb.Empty{})
			st, _ := status.FromError(err)

			assert.Equal(t, tt.wantStatus, st.Code())
			if tt.wantStatus == codes.OK {
				assert.Len(t, resp.Url, tt.wantCount)
				for _, u := range resp.Url {
					assert.Contains(t, u.ShortUrl, "http://localhost:8080")
					assert.Contains(t, u.OriginalUrl, "https://example.com")
				}
			}
		})
	}
}
