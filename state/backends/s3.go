package backends

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/schemabounce/kolumn/sdk/state"
	sdktypes "github.com/schemabounce/kolumn/sdk/types"
)

// S3Backend implements state storage in Amazon S3
type S3Backend struct {
	s3Client   *s3.Client
	config     *S3Config
	configured bool
}

// S3Config contains S3 backend configuration
type S3Config struct {
	Region               string `json:"region"`
	Bucket               string `json:"bucket"`
	KeyPrefix            string `json:"key_prefix"`
	Encrypt              bool   `json:"encrypt"`
	KMSKeyID             string `json:"kms_key_id"`
	ServerSideEncryption string `json:"server_side_encryption"`
	ACL                  string `json:"acl"`
	StorageClass         string `json:"storage_class"`
	MaxRetries           int    `json:"max_retries"`
	SkipCredentials      bool   `json:"skip_credentials"`
	Profile              string `json:"profile"`
	AccessKey            string `json:"access_key"`
	SecretKey            string `json:"secret_key"`
	SessionToken         string `json:"session_token"`
	Endpoint             string `json:"endpoint"` // For S3-compatible services
	ForcePathStyle       bool   `json:"force_path_style"`
}

// NewS3Backend creates a new S3 backend
func NewS3Backend() *S3Backend {
	return &S3Backend{}
}

// Configure sets up the S3 backend
func (b *S3Backend) Configure(ctx context.Context, config map[string]interface{}) error {
	// Parse configuration
	s3Config, err := parseS3Config(config)
	if err != nil {
		return fmt.Errorf("invalid S3 configuration: %w", err)
	}

	b.config = s3Config

	// Load AWS configuration
	var cfg aws.Config
	if s3Config.SkipCredentials {
		cfg, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(s3Config.Region),
			awsconfig.WithCredentialsProvider(aws.AnonymousCredentials{}),
		)
	} else {
		cfg, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(s3Config.Region),
		)
	}

	if err != nil {
		return fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// Override credentials if provided
	if s3Config.AccessKey != "" && s3Config.SecretKey != "" {
		cfg.Credentials = aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     s3Config.AccessKey,
				SecretAccessKey: s3Config.SecretKey,
				SessionToken:    s3Config.SessionToken,
			}, nil
		})
	}

	// Create S3 client
	var s3Options []func(*s3.Options)
	if s3Config.Endpoint != "" {
		s3Options = append(s3Options, s3.WithEndpointResolver(
			s3.EndpointResolverFunc(func(region string, options s3.EndpointResolverOptions) (aws.Endpoint, error) {
				return aws.Endpoint{URL: s3Config.Endpoint}, nil
			}),
		))
	}

	if s3Config.ForcePathStyle {
		s3Options = append(s3Options, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	if s3Config.MaxRetries > 0 {
		s3Options = append(s3Options, func(o *s3.Options) {
			o.RetryMaxAttempts = s3Config.MaxRetries
		})
	}

	b.s3Client = s3.NewFromConfig(cfg, s3Options...)

	// Validate bucket access
	if err := b.validateBucket(ctx); err != nil {
		return fmt.Errorf("bucket validation failed: %w", err)
	}

	b.configured = true
	return nil
}

// GetState retrieves state by name
func (b *S3Backend) GetState(ctx context.Context, name string) (*sdktypes.UniversalState, error) {
	if !b.configured {
		return nil, fmt.Errorf("backend not configured")
	}

	key := b.getStateKey(name)

	result, err := b.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.config.Bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		if b.isNoSuchKeyError(err) {
			return nil, fmt.Errorf("state '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to load state from S3: %w", err)
	}
	defer result.Body.Close()

	// Read state data
	stateData, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read state data: %w", err)
	}

	// Parse state JSON
	var st sdktypes.UniversalState
	if err := json.Unmarshal(stateData, &st); err != nil {
		return nil, fmt.Errorf("failed to parse state JSON: %w", err)
	}

	// Set timestamps from S3 metadata if available
	if result.LastModified != nil {
		st.UpdatedAt = *result.LastModified
	}

	return &st, nil
}

// PutState stores state by name
func (b *S3Backend) PutState(ctx context.Context, name string, st *sdktypes.UniversalState) error {
	if !b.configured {
		return fmt.Errorf("backend not configured")
	}

	if st == nil {
		return fmt.Errorf("state cannot be nil")
	}

	key := b.getStateKey(name)

	// Update timestamp
	st.UpdatedAt = time.Now()

	// Serialize state to JSON
	stateData, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}

	// Prepare put object input
	input := &s3.PutObjectInput{
		Bucket:      aws.String(b.config.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(stateData),
		ContentType: aws.String("application/json"),
		Metadata: map[string]string{
			"kolumn-version": st.TerraformVersion,
			"serial":         fmt.Sprintf("%d", st.Serial),
			"lineage":        st.Lineage,
		},
	}

	// Set server-side encryption
	if b.config.Encrypt {
		if b.config.KMSKeyID != "" {
			input.ServerSideEncryption = types.ServerSideEncryptionAwsKms
			input.SSEKMSKeyId = aws.String(b.config.KMSKeyID)
		} else {
			input.ServerSideEncryption = types.ServerSideEncryptionAes256
		}
	}

	// Set ACL
	if b.config.ACL != "" {
		input.ACL = types.ObjectCannedACL(b.config.ACL)
	}

	// Set storage class
	if b.config.StorageClass != "" {
		input.StorageClass = types.StorageClass(b.config.StorageClass)
	}

	_, err = b.s3Client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to save state to S3: %w", err)
	}

	return nil
}

// DeleteState removes state by name
func (b *S3Backend) DeleteState(ctx context.Context, name string) error {
	if !b.configured {
		return fmt.Errorf("backend not configured")
	}

	key := b.getStateKey(name)

	_, err := b.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(b.config.Bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete state from S3: %w", err)
	}

	return nil
}

// ListStates lists all available states
func (b *S3Backend) ListStates(ctx context.Context) ([]string, error) {
	if !b.configured {
		return nil, fmt.Errorf("backend not configured")
	}

	prefix := b.config.KeyPrefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	result, err := b.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(b.config.Bucket),
		Prefix: aws.String(prefix),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list states: %w", err)
	}

	var states []string
	for _, obj := range result.Contents {
		if obj.Key == nil {
			continue
		}

		key := *obj.Key

		// Remove prefix
		if prefix != "" {
			key = strings.TrimPrefix(key, prefix)
		}

		// Remove .klstate extension if present
		if strings.HasSuffix(key, ".klstate") {
			key = strings.TrimSuffix(key, ".klstate")
		}

		if key != "" {
			states = append(states, key)
		}
	}

	return states, nil
}

// Lock acquires a lock on the state
// Note: S3 doesn't provide native locking, so this is a no-op
// For production use, you would need DynamoDB for locking
func (b *S3Backend) Lock(ctx context.Context, info *state.LockInfo) (string, error) {
	if !b.configured {
		return "", fmt.Errorf("backend not configured")
	}

	if info == nil {
		return "", fmt.Errorf("lock info cannot be nil")
	}

	// S3 doesn't provide native locking without DynamoDB
	// For this simplified implementation, we'll just return success
	// In production, you should configure DynamoDB for proper locking
	return info.ID, nil
}

// Unlock releases a lock on the state
// Note: S3 doesn't provide native locking, so this is a no-op
func (b *S3Backend) Unlock(ctx context.Context, lockID string, info *state.LockInfo) error {
	if !b.configured {
		return fmt.Errorf("backend not configured")
	}

	if info == nil {
		return fmt.Errorf("lock info cannot be nil")
	}

	// S3 doesn't provide native locking without DynamoDB
	// For this simplified implementation, we'll just return success
	return nil
}

// Helper methods

func (b *S3Backend) validateBucket(ctx context.Context) error {
	// Check if bucket exists and is accessible
	_, err := b.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(b.config.Bucket),
	})
	if err != nil {
		return fmt.Errorf("bucket '%s' not accessible: %w", b.config.Bucket, err)
	}

	return nil
}

func (b *S3Backend) getStateKey(name string) string {
	key := name
	if !strings.HasSuffix(key, ".klstate") {
		key += ".klstate"
	}

	if b.config.KeyPrefix != "" {
		prefix := b.config.KeyPrefix
		if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		key = prefix + key
	}

	return key
}

func (b *S3Backend) isNoSuchKeyError(err error) bool {
	if err == nil {
		return false
	}

	// Check for NoSuchKey error
	errStr := err.Error()
	return strings.Contains(errStr, "NoSuchKey") || strings.Contains(errStr, "NotFound")
}

func parseS3Config(config map[string]interface{}) (*S3Config, error) {
	cfg := &S3Config{
		Region:       "us-east-1",
		MaxRetries:   3,
		StorageClass: "STANDARD",
		KeyPrefix:    "",
	}

	// Parse configuration map
	if region, ok := config["region"].(string); ok {
		cfg.Region = region
	}

	if bucket, ok := config["bucket"].(string); ok {
		cfg.Bucket = bucket
	}

	if keyPrefix, ok := config["key_prefix"].(string); ok {
		cfg.KeyPrefix = keyPrefix
	}

	if encrypt, ok := config["encrypt"].(bool); ok {
		cfg.Encrypt = encrypt
	}

	if kmsKeyID, ok := config["kms_key_id"].(string); ok {
		cfg.KMSKeyID = kmsKeyID
	}

	if serverSideEncryption, ok := config["server_side_encryption"].(string); ok {
		cfg.ServerSideEncryption = serverSideEncryption
	}

	if acl, ok := config["acl"].(string); ok {
		cfg.ACL = acl
	}

	if storageClass, ok := config["storage_class"].(string); ok {
		cfg.StorageClass = storageClass
	}

	if maxRetries, ok := config["max_retries"].(float64); ok {
		cfg.MaxRetries = int(maxRetries)
	} else if maxRetries, ok := config["max_retries"].(int); ok {
		cfg.MaxRetries = maxRetries
	}

	if skipCredentials, ok := config["skip_credentials"].(bool); ok {
		cfg.SkipCredentials = skipCredentials
	}

	if profile, ok := config["profile"].(string); ok {
		cfg.Profile = profile
	}

	if accessKey, ok := config["access_key"].(string); ok {
		cfg.AccessKey = accessKey
	}

	if secretKey, ok := config["secret_key"].(string); ok {
		cfg.SecretKey = secretKey
	}

	if sessionToken, ok := config["session_token"].(string); ok {
		cfg.SessionToken = sessionToken
	}

	if endpoint, ok := config["endpoint"].(string); ok {
		cfg.Endpoint = endpoint
	}

	if forcePathStyle, ok := config["force_path_style"].(bool); ok {
		cfg.ForcePathStyle = forcePathStyle
	}

	// Validate required fields
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	return cfg, nil
}
