package provider

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider implements Provider interface for testing
type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Call(ctx context.Context, req *Request) (*Response, error) {
	return &Response{Content: "mock response"}, nil
}

// Helper to clear registry between tests
func clearRegistry() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]func() (Provider, error))
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		factory      func() (Provider, error)
	}{
		{
			name:         "register single provider",
			providerName: "test-provider",
			factory: func() (Provider, error) {
				return &mockProvider{name: "test-provider"}, nil
			},
		},
		{
			name:         "register with different name",
			providerName: "another-provider",
			factory: func() (Provider, error) {
				return &mockProvider{name: "another-provider"}, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearRegistry()

			Register(tt.providerName, tt.factory)
			assert.True(t, IsRegistered(tt.providerName))
		})
	}
}

func TestRegister_Overwrite(t *testing.T) {
	clearRegistry()

	// Register first factory
	Register("test", func() (Provider, error) {
		return &mockProvider{name: "first"}, nil
	})

	// Register second factory with same name
	Register("test", func() (Provider, error) {
		return &mockProvider{name: "second"}, nil
	})

	// Get should return the second factory
	p, err := Get("test")
	require.NoError(t, err)
	assert.Equal(t, "second", p.Name())
}

func TestGet(t *testing.T) {
	tests := []struct {
		name         string
		setup        func()
		providerName string
		wantErr      bool
		wantName     string
	}{
		{
			name: "get existing provider",
			setup: func() {
				Register("existing", func() (Provider, error) {
					return &mockProvider{name: "existing"}, nil
				})
			},
			providerName: "existing",
			wantErr:      false,
			wantName:     "existing",
		},
		{
			name:         "get unknown provider",
			setup:        func() {},
			providerName: "unknown",
			wantErr:      true,
		},
		{
			name: "factory returns error",
			setup: func() {
				Register("error-factory", func() (Provider, error) {
					return nil, errors.New("factory error")
				})
			},
			providerName: "error-factory",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearRegistry()
			tt.setup()

			p, err := Get(tt.providerName)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantName, p.Name())
		})
	}
}

func TestGet_ErrorIncludesAvailable(t *testing.T) {
	clearRegistry()

	Register("provider-a", func() (Provider, error) {
		return &mockProvider{name: "provider-a"}, nil
	})
	Register("provider-b", func() (Provider, error) {
		return &mockProvider{name: "provider-b"}, nil
	})

	_, err := Get("unknown")
	require.Error(t, err)

	errStr := err.Error()
	assert.Contains(t, errStr, "unknown")
	assert.Contains(t, errStr, "provider-a")
	assert.Contains(t, errStr, "provider-b")
}

func TestAvailable(t *testing.T) {
	tests := []struct {
		name      string
		setup     func()
		wantCount int
	}{
		{
			name:      "empty registry",
			setup:     func() {},
			wantCount: 0,
		},
		{
			name: "single provider",
			setup: func() {
				Register("single", func() (Provider, error) {
					return &mockProvider{}, nil
				})
			},
			wantCount: 1,
		},
		{
			name: "multiple providers",
			setup: func() {
				Register("one", func() (Provider, error) { return &mockProvider{}, nil })
				Register("two", func() (Provider, error) { return &mockProvider{}, nil })
				Register("three", func() (Provider, error) { return &mockProvider{}, nil })
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearRegistry()
			tt.setup()

			available := Available()
			assert.Len(t, available, tt.wantCount)
		})
	}
}

func TestIsRegistered(t *testing.T) {
	tests := []struct {
		name         string
		setup        func()
		providerName string
		want         bool
	}{
		{
			name: "registered provider",
			setup: func() {
				Register("registered", func() (Provider, error) {
					return &mockProvider{}, nil
				})
			},
			providerName: "registered",
			want:         true,
		},
		{
			name:         "unregistered provider",
			setup:        func() {},
			providerName: "unregistered",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearRegistry()
			tt.setup()

			got := IsRegistered(tt.providerName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	clearRegistry()

	// Register initial provider
	Register("concurrent", func() (Provider, error) {
		return &mockProvider{name: "concurrent"}, nil
	})

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent reads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = Get("concurrent")
			_ = Available()
			_ = IsRegistered("concurrent")
		}()
	}

	// Concurrent writes
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			Register("concurrent", func() (Provider, error) {
				return &mockProvider{name: "concurrent"}, nil
			})
		}(i)
	}

	wg.Wait()

	// Should not panic and registry should still work
	assert.True(t, IsRegistered("concurrent"))
}
