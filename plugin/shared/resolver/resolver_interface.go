package resolver

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"github.com/safing/portmaster/plugin/shared/proto"
	"google.golang.org/grpc"
)

type (
	Resolver interface {
		Resolve(ctx context.Context, question *proto.DNSQuestion, connection *proto.Connection) (*proto.DNSResponse, error)
	}

	Plugin struct {
		plugin.NetRPCUnsupportedPlugin

		Impl Resolver
	}
)

// GRPCServer implements the gRPC server side of resolver.Service and implements plugin.GRPCPlugin.
func (p *Plugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterResolverServiceServer(s, &gRPCServer{
		Impl: p.Impl,
	})

	return nil
}

// GRPCClient implements the gRPC client side of resolver.Service and implements plugin.GRPCPlugin.
func (p *Plugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &gRPCClient{
		client: proto.NewResolverServiceClient(c),
		broker: broker,
	}, nil
}

var _ plugin.GRPCPlugin = &Plugin{}