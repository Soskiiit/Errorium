package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
)

// ErrNotFound is returned when an object is not found in the storage.
var ErrNotFound = errors.New("object not found")

// AvatarStore defines the storage contract for user avatars.
type AvatarStore interface {
	Put(ctx context.Context, key, contentType string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, string, error)
}

// S3ClientAPI defines the subset of S3 client operations needed by S3Storage.
type S3ClientAPI interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// S3Storage implements AvatarStore on top of Garage / AWS S3.
type S3Storage struct {
	client S3ClientAPI
	bucket string
}

// S3Store is an alias for S3Storage.
type S3Store = S3Storage

var _ AvatarStore = (*S3Storage)(nil)
var _ AvatarStore = (*MemoryStore)(nil)

// New constructs an S3Storage adapter using the given S3 configuration.
func New(cfg config.S3Config) (*S3Storage, error) {
	return NewS3Storage(cfg)
}

// NewS3Storage constructs an S3Storage adapter using the given S3 configuration.
func NewS3Storage(cfg config.S3Config) (*S3Storage, error) {
	accessKey := cfg.AccessKey
	if accessKey == "" {
		accessKey = cfg.AccessKeyID
	}
	secretKey := cfg.SecretKey
	if secretKey == "" {
		secretKey = cfg.SecretAccessKey
	}
	bucket := cfg.BucketName
	if bucket == "" {
		bucket = cfg.Bucket
	}

	endpoint := cfg.Endpoint
	if endpoint != "" && !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		scheme := "http"
		if cfg.UseSSL {
			scheme = "https"
		}
		endpoint = fmt.Sprintf("%s://%s", scheme, endpoint)
	}

	region := cfg.Region
	if region == "" {
		region = "garage"
	}

	optFns := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), optFns...)
	if err != nil {
		return nil, fmt.Errorf("storage: failed to load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})

	return NewWithClient(client, bucket), nil
}

// NewWithClient constructs an S3Storage adapter with a pre-configured S3ClientAPI implementation.
func NewWithClient(client S3ClientAPI, bucket string) *S3Storage {
	return &S3Storage{
		client: client,
		bucket: bucket,
	}
}

// Put writes an object with the specified key, content type, and payload data.
func (s *S3Storage) Put(ctx context.Context, key, contentType string, data []byte) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("storage: failed to put object: %w", err)
	}
	return nil
}

// Get retrieves an object by key, returning its data bytes and content type.
func (s *S3Storage) Get(ctx context.Context, key string) ([]byte, string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	output, err := s.client.GetObject(ctx, input)
	if err != nil {
		var nsk *types.NoSuchKey
		var nf *types.NotFound
		if errors.As(err, &nsk) || errors.As(err, &nf) {
			return nil, "", ErrNotFound
		}
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			code := apiErr.ErrorCode()
			if code == "NoSuchKey" || code == "NotFound" || code == "404" {
				return nil, "", ErrNotFound
			}
		}
		if strings.Contains(err.Error(), "NoSuchKey") || strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "404") {
			return nil, "", ErrNotFound
		}
		return nil, "", fmt.Errorf("storage: failed to get object: %w", err)
	}
	defer output.Body.Close()

	data, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, "", fmt.Errorf("storage: failed to read object body: %w", err)
	}

	contentType := ""
	if output.ContentType != nil && *output.ContentType != "" {
		contentType = *output.ContentType
	} else {
		contentType = http.DetectContentType(data)
	}

	return data, contentType, nil
}

// MemoryStore provides an in-memory implementation of AvatarStore for testing and offline fallback.
type MemoryStore struct {
	mu           sync.RWMutex
	data         map[string][]byte
	contentTypes map[string]string
}

// NewMemoryStore initializes an empty in-memory avatar store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data:         make(map[string][]byte),
		contentTypes: make(map[string]string),
	}
}

// Put saves data in memory.
func (m *MemoryStore) Put(ctx context.Context, key, contentType string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cp := make([]byte, len(data))
	copy(cp, data)
	m.data[key] = cp
	m.contentTypes[key] = contentType
	return nil
}

// Get reads data from memory.
func (m *MemoryStore) Get(ctx context.Context, key string) ([]byte, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, ok := m.data[key]
	if !ok {
		return nil, "", ErrNotFound
	}

	ct := m.contentTypes[key]
	if ct == "" {
		ct = http.DetectContentType(data)
	}

	cp := make([]byte, len(data))
	copy(cp, data)
	return cp, ct, nil
}
