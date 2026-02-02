package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

type MockAdapter struct {
	Config MockConfig
}

type MockConfig struct {
	StdoutJSON     string
	ExitCode       int
	TokensUsed     int
	SimulatedDelay time.Duration
	ShouldFail     bool
	FailError      error
}

type MockOption func(*MockConfig)

func WithStdoutJSON(stdout string) MockOption {
	return func(c *MockConfig) {
		c.StdoutJSON = stdout
	}
}

func WithExitCode(code int) MockOption {
	return func(c *MockConfig) {
		c.ExitCode = code
	}
}

func WithTokensUsed(tokens int) MockOption {
	return func(c *MockConfig) {
		c.TokensUsed = tokens
	}
}

func WithSimulatedDelay(delay time.Duration) MockOption {
	return func(c *MockConfig) {
		c.SimulatedDelay = delay
	}
}

func WithFailure(err error) MockOption {
	return func(c *MockConfig) {
		c.ShouldFail = true
		c.FailError = err
	}
}

func NewMockAdapter(opts ...MockOption) *MockAdapter {
	cfg := MockConfig{
		ExitCode:       0,
		TokensUsed:     100,
		SimulatedDelay: 0,
		ShouldFail:     false,
		StdoutJSON:     `{"result": "success"}`,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &MockAdapter{Config: cfg}
}

func (m *MockAdapter) Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error) {
	if m.Config.SimulatedDelay > 0 {
		select {
		case <-time.After(m.Config.SimulatedDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if m.Config.ShouldFail {
		return nil, m.Config.FailError
	}

	stdout := m.Config.StdoutJSON
	if stdout == "" {
		stdout = fmt.Sprintf(`{"adapter": "%s", "persona": "%s", "prompt_length": %d}`,
			cfg.Adapter, cfg.Persona, len(cfg.Prompt))
	}

	var artifacts []string
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &parsed); err == nil {
		if artifactList, ok := parsed["artifacts"].([]interface{}); ok {
			for _, a := range artifactList {
				if s, ok := a.(string); ok {
					artifacts = append(artifacts, s)
				}
			}
		}
	}

	tokens := m.Config.TokensUsed
	if tokens == 0 {
		tokens = len(cfg.Prompt) / 4
	}

	return &AdapterResult{
		ExitCode:   m.Config.ExitCode,
		Stdout:     bytes.NewReader([]byte(stdout)),
		TokensUsed: tokens,
		Artifacts:  artifacts,
	}, nil
}

type MockAdapterRegistry struct {
	mu       sync.RWMutex
	adapters map[string]*MockAdapter
}

func NewMockAdapterRegistry() *MockAdapterRegistry {
	return &MockAdapterRegistry{
		adapters: make(map[string]*MockAdapter),
	}
}

func (r *MockAdapterRegistry) Register(name string, adapter *MockAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[name] = adapter
}

func (r *MockAdapterRegistry) Get(name string) *MockAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.adapters[name]
}

func (r *MockAdapterRegistry) CreateRunner(name string) AdapterRunner {
	adapter := r.Get(name)
	if adapter == nil {
		adapter = NewMockAdapter()
	}
	return &registeredRunner{
		registry: r,
		name:     name,
	}
}

type registeredRunner struct {
	registry *MockAdapterRegistry
	name     string
}

func (r *registeredRunner) Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error) {
	adapter := r.registry.Get(r.name)
	if adapter == nil {
		adapter = NewMockAdapter()
	}
	cfg.Adapter = r.name
	return adapter.Run(ctx, cfg)
}

type SlowReader struct {
	data      []byte
	readPos   int
	chunkSize int
	delay     time.Duration
	mu        sync.Mutex
}

func NewSlowReader(data string, chunkSize int, delay time.Duration) *SlowReader {
	return &SlowReader{
		data:      []byte(data),
		readPos:   0,
		chunkSize: chunkSize,
		delay:     delay,
	}
}

func (r *SlowReader) Read(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.readPos >= len(r.data) {
		return 0, io.EOF
	}

	remaining := len(r.data) - r.readPos
	toRead := r.chunkSize
	if toRead > remaining {
		toRead = remaining
	}
	if toRead > len(p) {
		toRead = len(p)
	}

	time.Sleep(r.delay)

	copy(p, r.data[r.readPos:r.readPos+toRead])
	r.readPos += toRead

	if r.readPos >= len(r.data) {
		return toRead, io.EOF
	}

	return toRead, nil
}
