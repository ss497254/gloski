package agent

import (
	"context"
	"net"
	"net/url"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

func dialMaster(ctx context.Context, raw string) (*grpc.ClientConn, error) {
	raw = strings.TrimSpace(raw)
	credentials := insecure.NewCredentials()
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(32<<20), grpc.MaxCallSendMsgSize(32<<20)),
		// Pings notice a dead path (sleeping laptop, dropped tunnel) within
		// about 30 seconds instead of waiting on TCP timeouts.
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                20 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	target := raw
	if strings.HasPrefix(raw, "unix://") {
		socket := strings.TrimPrefix(raw, "unix://")
		target = "unix://" + socket
		options = append(options, grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		}))
	} else if parsed, err := url.Parse(raw); err == nil && parsed.Host != "" {
		target = parsed.Host
	}
	return grpc.DialContext(ctx, target, options...)
}
