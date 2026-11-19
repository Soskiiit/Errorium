package storage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
	"github.com/soskiiit/errorium/services/gateway/pkg/storage"
)

type mockS3Client struct {
	putFunc func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	getFunc func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

func (m *mockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.putFunc != nil {
		return m.putFunc(ctx, params, optFns...)
	}
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, params, optFns...)
	}
	return &s3.GetObjectOutput{
		Body:        io.NopCloser(bytes.NewReader([]byte("test-data"))),
		ContentType: aws.String("image/png"),
	}, nil
}

func TestAvatarStore_InterfaceCompliance(t *testing.T) {
	var _ storage.AvatarStore = (*storage.S3Storage)(nil)
	var _ storage.AvatarStore = (*storage.MemoryStore)(nil)
}

func TestS3Storage_Put_Success(t *testing.T) {
	var capturedBucket, capturedKey, capturedContentType string
	var capturedData []byte

	client := &mockS3Client{
		putFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			capturedBucket = aws.ToString(params.Bucket)
			capturedKey = aws.ToString(params.Key)
			capturedContentType = aws.ToString(params.ContentType)
			data, _ := io.ReadAll(params.Body)
			capturedData = data
			return &s3.PutObjectOutput{}, nil
		},
	}

	store := storage.NewWithClient(client, "test-bucket")
	err := store.Put(context.Background(), "avatars/123", "image/png", []byte("image-data"))
	if err != nil {
		t.Fatalf("expected Put to succeed, got %v", err)
	}

	if capturedBucket != "test-bucket" {
		t.Errorf("expected bucket 'test-bucket', got %q", capturedBucket)
	}
	if capturedKey != "avatars/123" {
		t.Errorf("expected key 'avatars/123', got %q", capturedKey)
	}
	if capturedContentType != "image/png" {
		t.Errorf("expected content type 'image/png', got %q", capturedContentType)
	}
	if string(capturedData) != "image-data" {
		t.Errorf("expected data 'image-data', got %q", string(capturedData))
	}
}

func TestS3Storage_Put_Error(t *testing.T) {
	client := &mockS3Client{
		putFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			return nil, errors.New("s3 connection timeout")
		},
	}

	store := storage.NewWithClient(client, "test-bucket")
	err := store.Put(context.Background(), "avatars/123", "image/png", []byte("image-data"))
	if err == nil {
		t.Fatalf("expected Put to return error, got nil")
	}
}

func TestS3Storage_Get_Success(t *testing.T) {
	client := &mockS3Client{
		getFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return &s3.GetObjectOutput{
				Body:        io.NopCloser(bytes.NewReader([]byte("png-bytes"))),
				ContentType: aws.String("image/png"),
			}, nil
		},
	}

	store := storage.NewWithClient(client, "test-bucket")
	data, ct, err := store.Get(context.Background(), "avatars/123")
	if err != nil {
		t.Fatalf("expected Get to succeed, got %v", err)
	}

	if string(data) != "png-bytes" {
		t.Errorf("expected 'png-bytes', got %q", string(data))
	}
	if ct != "image/png" {
		t.Errorf("expected 'image/png', got %q", ct)
	}
}

func TestS3Storage_Get_ContentTypeDetectionFallback(t *testing.T) {
	// Minimal 1x1 PNG bytes so DetectContentType identifies it as image/png.
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	client := &mockS3Client{
		getFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return &s3.GetObjectOutput{
				Body:        io.NopCloser(bytes.NewReader(pngHeader)),
				ContentType: nil, // no content type provided by S3
			}, nil
		},
	}

	store := storage.NewWithClient(client, "test-bucket")
	data, ct, err := store.Get(context.Background(), "avatars/123")
	if err != nil {
		t.Fatalf("expected Get to succeed, got %v", err)
	}

	if len(data) != len(pngHeader) {
		t.Errorf("expected data len %d, got %d", len(pngHeader), len(data))
	}
	if ct != "image/png" {
		t.Errorf("expected detected Content-Type 'image/png', got %q", ct)
	}
}

func TestS3Storage_Get_NotFound_NoSuchKey(t *testing.T) {
	client := &mockS3Client{
		getFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return nil, &types.NoSuchKey{}
		},
	}

	store := storage.NewWithClient(client, "test-bucket")
	_, _, err := store.Get(context.Background(), "avatars/unknown")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for NoSuchKey, got %v", err)
	}
}

func TestS3Storage_Get_NotFound_APIError(t *testing.T) {
	client := &mockS3Client{
		getFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return nil, &smithy.GenericAPIError{
				Code:    "NotFound",
				Message: "object not found in bucket",
			}
		},
	}

	store := storage.NewWithClient(client, "test-bucket")
	_, _, err := store.Get(context.Background(), "avatars/unknown")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for NotFound APIError, got %v", err)
	}
}

func TestS3Storage_Get_OtherError(t *testing.T) {
	client := &mockS3Client{
		getFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return nil, errors.New("connection reset by peer")
		},
	}

	store := storage.NewWithClient(client, "test-bucket")
	_, _, err := store.Get(context.Background(), "avatars/unknown")
	if errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("did not expect ErrNotFound for generic network error, got %v", err)
	}
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestS3Storage_Constructor(t *testing.T) {
	cfg := config.S3Config{
		Endpoint:   "localhost:3900",
		AccessKey:  "garageadmin",
		SecretKey:  "garageadminpassword",
		BucketName: "errorium-avatars",
		Region:     "garage",
		UseSSL:     false,
	}

	store, err := storage.NewS3Storage(cfg)
	if err != nil {
		t.Fatalf("expected NewS3Storage to succeed, got %v", err)
	}
	if store == nil {
		t.Fatalf("expected non-nil store")
	}

	store2, err := storage.New(cfg)
	if err != nil {
		t.Fatalf("expected New alias to succeed, got %v", err)
	}
	if store2 == nil {
		t.Fatalf("expected non-nil store2")
	}
}

func TestMemoryStore_PutAndGet(t *testing.T) {
	store := storage.NewMemoryStore()

	ctx := context.Background()
	_, _, err := store.Get(ctx, "avatars/1")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing key, got %v", err)
	}

	err = store.Put(ctx, "avatars/1", "image/png", []byte("avatar-content"))
	if err != nil {
		t.Fatalf("expected Put to succeed, got %v", err)
	}

	data, ct, err := store.Get(ctx, "avatars/1")
	if err != nil {
		t.Fatalf("expected Get to succeed, got %v", err)
	}
	if string(data) != "avatar-content" {
		t.Errorf("expected 'avatar-content', got %q", string(data))
	}
	if ct != "image/png" {
		t.Errorf("expected 'image/png', got %q", ct)
	}
}
