package tracing

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestUnaryClientUsesConfiguredTimeout(t *testing.T) {
	conn, err := grpc.NewClient("passthrough:///unused", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	timeout := 15 * time.Second
	start := time.Now()
	invoke := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		deadline, ok := ctx.Deadline()
		if !ok || deadline.Before(start.Add(timeout)) || deadline.After(time.Now().Add(timeout)) {
			t.Fatalf("deadline = %v, want approximately %v", deadline, start.Add(timeout))
		}
		return nil
	}
	if err := UnaryClientInterceptorWithTimeout(timeout)(context.Background(), "/auth.v1.AuthService/Login", nil, nil, conn, invoke); err != nil {
		t.Fatal(err)
	}
}

func TestUnaryClientRetriesOnlyReads(t *testing.T) {
	conn, err := grpc.NewClient("passthrough:///unused", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	for _, tc := range []struct {
		method    string
		wantCalls int
	}{
		{"/accounts.v1.AccountService/GetAccount", 2},
		{"/transactions.v1.TransactionService/Transfer", 1},
	} {
		t.Run(tc.method, func(t *testing.T) {
			calls := 0
			invoke := func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
				calls++
				if calls == 1 {
					return status.Error(codes.Unavailable, "temporary outage")
				}
				return nil
			}
			_ = UnaryClientInterceptor()(context.Background(), tc.method, nil, nil, conn, invoke)
			if calls != tc.wantCalls {
				t.Fatalf("calls = %d, want %d", calls, tc.wantCalls)
			}
		})
	}
}
