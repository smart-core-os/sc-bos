package protopkg

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestNewToOldInterceptor_Unary(t *testing.T) {
	interceptor := NewNewToOldInterceptor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "new format translated",
			input:    "/smartcore.bos.meter.v1.MeterApi/GetMeterReading",
			expected: "/smartcore.bos.MeterApi/GetMeterReading",
		},
		{
			name:     "old format unchanged",
			input:    "/smartcore.bos.MeterApi/GetMeterReading",
			expected: "/smartcore.bos.MeterApi/GetMeterReading",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()

			handler := func(_ context.Context, _ any) (any, error) {
				return nil, nil
			}

			info := &grpc.UnaryServerInfo{
				FullMethod: tt.input,
			}

			_, _ = interceptor.UnaryInterceptor()(ctx, nil, info, handler)

			if info.FullMethod != tt.expected {
				t.Errorf("method = %q, want %q", info.FullMethod, tt.expected)
			}
		})
	}
}

func TestNewToOldInterceptor_Stream(t *testing.T) {
	interceptor := NewNewToOldInterceptor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "new format translated",
			input:    "/smartcore.bos.alert.v1.AlertApi/PullAlerts",
			expected: "/smartcore.bos.AlertApi/PullAlerts",
		},
		{
			name:     "old format unchanged",
			input:    "/smartcore.bos.AlertApi/PullAlerts",
			expected: "/smartcore.bos.AlertApi/PullAlerts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(_ any, _ grpc.ServerStream) error {
				return nil
			}

			info := &grpc.StreamServerInfo{
				FullMethod: tt.input,
			}

			ss := &mockServerStream{ctx: context.Background()}
			_ = interceptor.StreamInterceptor()(nil, ss, info, handler)

			if info.FullMethod != tt.expected {
				t.Errorf("method = %q, want %q", info.FullMethod, tt.expected)
			}
		})
	}
}

func TestOldToNewInterceptor_Unary(t *testing.T) {
	interceptor := NewOldToNewInterceptor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "old format translated",
			input:    "/smartcore.bos.MeterApi/GetMeterReading",
			expected: "/smartcore.bos.meter.v1.MeterApi/GetMeterReading",
		},
		{
			name:     "new format unchanged",
			input:    "/smartcore.bos.meter.v1.MeterApi/GetMeterReading",
			expected: "/smartcore.bos.meter.v1.MeterApi/GetMeterReading",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()

			handler := func(_ context.Context, _ any) (any, error) {
				return nil, nil
			}

			info := &grpc.UnaryServerInfo{
				FullMethod: tt.input,
			}

			_, _ = interceptor.UnaryInterceptor()(ctx, nil, info, handler)

			if info.FullMethod != tt.expected {
				t.Errorf("method = %q, want %q", info.FullMethod, tt.expected)
			}
		})
	}
}

func TestOldToNewInterceptor_Stream(t *testing.T) {
	interceptor := NewOldToNewInterceptor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "old format translated",
			input:    "/smartcore.bos.AlertApi/PullAlerts",
			expected: "/smartcore.bos.alert.v1.AlertApi/PullAlerts",
		},
		{
			name:     "new format unchanged",
			input:    "/smartcore.bos.alert.v1.AlertApi/PullAlerts",
			expected: "/smartcore.bos.alert.v1.AlertApi/PullAlerts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(_ any, _ grpc.ServerStream) error {
				return nil
			}

			info := &grpc.StreamServerInfo{
				FullMethod: tt.input,
			}

			ss := &mockServerStream{ctx: context.Background()}
			_ = interceptor.StreamInterceptor()(nil, ss, info, handler)

			if info.FullMethod != tt.expected {
				t.Errorf("method = %q, want %q", info.FullMethod, tt.expected)
			}
		})
	}
}

// TestInterceptorWithUnknownServiceHandler tests that the interceptor does enough for our
// prod code, which uses an UnknownServiceHandler, to correctly see the translated method name.
func TestInterceptorWithUnknownServiceHandler(t *testing.T) {
	var capturedMethod string
	unknownHandler := func(_ any, ss grpc.ServerStream) error {
		method, ok := grpc.Method(ss.Context())
		if !ok {
			t.Error("grpc.Method returned false")
		}
		capturedMethod = method
		return nil
	}

	input := "/smartcore.bos.MeterApi/GetMeterReading"
	expected := "/smartcore.bos.meter.v1.MeterApi/GetMeterReading"
	interceptor := NewOldToNewInterceptor()

	server := grpc.NewServer(
		grpc.UnknownServiceHandler(unknownHandler),
		grpc.StreamInterceptor(interceptor.StreamInterceptor()),
	)

	lis := bufconn.Listen(1024 * 1024)
	go func() {
		_ = server.Serve(lis)
	}()
	defer server.Stop()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient() failed: %v", err)
	}
	defer conn.Close()

	_ = conn.Invoke(t.Context(), input, nil, nil)

	if capturedMethod != expected {
		t.Errorf("grpc.Method() = %q, want %q", capturedMethod, expected)
	}
}

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}
