package secret

import (
	"context"
	"errors"
	"testing"

	openbao "github.com/openbao/openbao/api/v2"
)

type fakeKV struct {
	data          map[string]map[string]interface{}
	deleteTargets []string
}

func (f *fakeKV) Get(ctx context.Context, secretPath string) (*openbao.KVSecret, error) {
	data, ok := f.data[secretPath]
	if !ok {
		return nil, openbao.ErrSecretNotFound
	}

	return &openbao.KVSecret{Data: cloneSecretData(data)}, nil
}

func (f *fakeKV) Put(ctx context.Context, secretPath string, data map[string]interface{}, opts ...openbao.KVOption) (*openbao.KVSecret, error) {
	if f.data == nil {
		f.data = map[string]map[string]interface{}{}
	}
	f.data[secretPath] = cloneSecretData(data)
	return &openbao.KVSecret{Data: cloneSecretData(data)}, nil
}

func (f *fakeKV) DeleteMetadata(ctx context.Context, secretPath string) error {
	f.deleteTargets = append(f.deleteTargets, secretPath)
	delete(f.data, secretPath)
	return nil
}

func TestOpenBaoStorePutStringMergesExistingData(t *testing.T) {
	t.Parallel()

	store := &OpenBaoStore{
		kv: &fakeKV{
			data: map[string]map[string]interface{}{
				"app/config": {
					"value": "old",
					"extra": "keep",
				},
			},
		},
	}

	ref, err := store.PutString(context.Background(), "app/config", "", "new")
	if err != nil {
		t.Fatalf("PutString returned error: %v", err)
	}
	if ref != "app/config#value" {
		t.Fatalf("unexpected ref: %q", ref)
	}

	got, err := store.GetString(context.Background(), "app/config#extra")
	if err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
	if got != "keep" {
		t.Fatalf("expected preserved field %q, got %q", "keep", got)
	}

	got, err = store.GetString(context.Background(), "app/config#value")
	if err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
	if got != "new" {
		t.Fatalf("expected updated field %q, got %q", "new", got)
	}
}

func TestOpenBaoStorePutStringCreatesMissingSecret(t *testing.T) {
	t.Parallel()

	store := &OpenBaoStore{kv: &fakeKV{}}

	ref, err := store.PutString(context.Background(), "app/config", "dsn", "postgres://demo")
	if err != nil {
		t.Fatalf("PutString returned error: %v", err)
	}
	if ref != "app/config#dsn" {
		t.Fatalf("unexpected ref: %q", ref)
	}

	got, err := store.GetString(context.Background(), ref)
	if err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
	if got != "postgres://demo" {
		t.Fatalf("expected created value, got %q", got)
	}
}

func TestOpenBaoStoreDeleteRemovesOnlySelectedKey(t *testing.T) {
	t.Parallel()

	kv := &fakeKV{
		data: map[string]map[string]interface{}{
			"app/config": {
				"dsn":   "postgres://demo",
				"token": "keep-me",
			},
		},
	}
	store := &OpenBaoStore{kv: kv}

	if err := store.Delete(context.Background(), "app/config#dsn"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	if len(kv.deleteTargets) != 0 {
		t.Fatalf("expected DeleteMetadata not to be called, got %v", kv.deleteTargets)
	}

	if _, err := store.GetString(context.Background(), "app/config#dsn"); err == nil {
		t.Fatal("expected deleted key lookup to fail")
	}

	got, err := store.GetString(context.Background(), "app/config#token")
	if err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
	if got != "keep-me" {
		t.Fatalf("expected remaining key to stay intact, got %q", got)
	}
}

func TestOpenBaoStoreDeleteLastKeyRemovesSecret(t *testing.T) {
	t.Parallel()

	kv := &fakeKV{
		data: map[string]map[string]interface{}{
			"app/config": {
				"value": "only",
			},
		},
	}
	store := &OpenBaoStore{kv: kv}

	if err := store.Delete(context.Background(), "app/config#value"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	if len(kv.deleteTargets) != 1 || kv.deleteTargets[0] != "app/config" {
		t.Fatalf("expected DeleteMetadata for app/config, got %v", kv.deleteTargets)
	}

	if _, err := store.GetString(context.Background(), "app/config#value"); !errors.Is(err, openbao.ErrSecretNotFound) {
		t.Fatalf("expected missing secret after delete, got %v", err)
	}
}

func cloneSecretData(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}
