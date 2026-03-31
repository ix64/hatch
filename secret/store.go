package secret

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	openbao "github.com/openbao/openbao/api/v2"
)

type Store interface {
	Ping(ctx context.Context) error
	GetString(ctx context.Context, ref string) (string, error)
	PutString(ctx context.Context, path, key, value string) (string, error)
	PutFields(ctx context.Context, path string, data map[string]string) error
	Delete(ctx context.Context, ref string) error
}

type Ref struct {
	Path string
	Key  string
}

type OpenBaoOptions struct {
	Address       string
	Token         string
	Namespace     string
	KVMount       string
	Insecure      bool
	CACertFile    string
	CAPath        string
	TLSServerName string
	Timeout       time.Duration
}

type OpenBaoStore struct {
	client *openbao.Client
	kv     openBaoKV
}

type openBaoKV interface {
	Get(ctx context.Context, secretPath string) (*openbao.KVSecret, error)
	Put(ctx context.Context, secretPath string, data map[string]interface{}, opts ...openbao.KVOption) (*openbao.KVSecret, error)
	DeleteMetadata(ctx context.Context, secretPath string) error
}

func NewOpenBaoStore(opts OpenBaoOptions) (Store, error) {
	cfg := openbao.DefaultConfig()
	if opts.Address != "" {
		cfg.Address = opts.Address
	}
	if opts.Timeout > 0 {
		cfg.Timeout = opts.Timeout
	}
	if opts.Insecure || opts.CACertFile != "" || opts.CAPath != "" || opts.TLSServerName != "" {
		if err := cfg.ConfigureTLS(&openbao.TLSConfig{
			CACert:        opts.CACertFile,
			CAPath:        opts.CAPath,
			TLSServerName: opts.TLSServerName,
			Insecure:      opts.Insecure,
		}); err != nil {
			return nil, fmt.Errorf("configure openbao tls failed: %w", err)
		}
	}

	client, err := openbao.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create openbao client failed: %w", err)
	}
	if opts.Token != "" {
		client.SetToken(opts.Token)
	}
	if opts.Namespace != "" {
		client.SetNamespace(opts.Namespace)
	}

	mount := opts.KVMount
	if mount == "" {
		mount = "secret"
	}

	return &OpenBaoStore{
		client: client,
		kv:     client.KVv2(mount),
	}, nil
}

func ParseRef(ref string) (Ref, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return Ref{}, fmt.Errorf("secret ref is empty")
	}

	path, key, found := strings.Cut(ref, "#")
	path = strings.Trim(path, "/")
	key = strings.TrimSpace(key)
	if path == "" {
		return Ref{}, fmt.Errorf("secret ref %q is missing path", ref)
	}
	if !found || key == "" {
		key = "value"
	}

	return Ref{Path: path, Key: key}, nil
}

func BuildRef(path, key string) string {
	path = strings.Trim(path, "/")
	key = strings.TrimSpace(key)
	if key == "" {
		key = "value"
	}
	return path + "#" + key
}

func (s *OpenBaoStore) Ping(ctx context.Context) error {
	_, err := s.client.Sys().HealthWithContext(ctx)
	if err != nil {
		return fmt.Errorf("openbao health check failed: %w", err)
	}
	return nil
}

func (s *OpenBaoStore) GetString(ctx context.Context, ref string) (string, error) {
	r, err := ParseRef(ref)
	if err != nil {
		return "", err
	}

	secret, err := s.kv.Get(ctx, r.Path)
	if err != nil {
		return "", fmt.Errorf("read openbao secret %q failed: %w", r.Path, err)
	}

	value, ok := secret.Data[r.Key]
	if !ok {
		return "", fmt.Errorf("secret key %q not found in %q", r.Key, r.Path)
	}

	switch v := value.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		return fmt.Sprint(v), nil
	}
}

func (s *OpenBaoStore) PutString(ctx context.Context, path, key, value string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		key = "value"
	}

	path = strings.Trim(path, "/")
	if path == "" {
		return "", fmt.Errorf("secret path is empty")
	}

	secretData, err := s.readSecretData(ctx, path)
	if err != nil {
		return "", err
	}
	secretData[key] = value

	if _, err := s.kv.Put(ctx, path, secretData); err != nil {
		return "", fmt.Errorf("write openbao secret %q failed: %w", path, err)
	}

	return BuildRef(path, key), nil
}

func (s *OpenBaoStore) PutFields(ctx context.Context, path string, data map[string]string) error {
	path = strings.Trim(path, "/")
	if path == "" {
		return fmt.Errorf("secret path is empty")
	}
	if len(data) == 0 {
		return fmt.Errorf("secret payload is empty")
	}

	payload := make(map[string]interface{}, len(data))
	for k, v := range data {
		payload[k] = v
	}

	if _, err := s.kv.Put(ctx, path, payload); err != nil {
		return fmt.Errorf("write openbao secret %q failed: %w", path, err)
	}
	return nil
}

func (s *OpenBaoStore) Delete(ctx context.Context, ref string) error {
	r, err := ParseRef(ref)
	if err != nil {
		return err
	}

	secretData, err := s.readSecretData(ctx, r.Path)
	if err != nil {
		return err
	}
	if _, ok := secretData[r.Key]; !ok {
		return fmt.Errorf("secret key %q not found in %q", r.Key, r.Path)
	}

	delete(secretData, r.Key)
	if len(secretData) == 0 {
		if err := s.kv.DeleteMetadata(ctx, r.Path); err != nil {
			return fmt.Errorf("delete openbao secret %q failed: %w", r.Path, err)
		}
		return nil
	}

	if _, err := s.kv.Put(ctx, r.Path, secretData); err != nil {
		return fmt.Errorf("write openbao secret %q failed: %w", r.Path, err)
	}
	return nil
}

func (s *OpenBaoStore) readSecretData(ctx context.Context, path string) (map[string]interface{}, error) {
	secret, err := s.kv.Get(ctx, path)
	switch {
	case err == nil:
	case errors.Is(err, openbao.ErrSecretNotFound):
		return map[string]interface{}{}, nil
	default:
		return nil, fmt.Errorf("read openbao secret %q failed: %w", path, err)
	}

	data := make(map[string]interface{}, len(secret.Data))
	for key, value := range secret.Data {
		data[key] = value
	}
	return data, nil
}
