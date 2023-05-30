// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.2
// source: resolver.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ResolverServiceClient is the client API for ResolverService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ResolverServiceClient interface {
	Resolve(ctx context.Context, in *ResolveRequest, opts ...grpc.CallOption) (*ResolveResponse, error)
}

type resolverServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewResolverServiceClient(cc grpc.ClientConnInterface) ResolverServiceClient {
	return &resolverServiceClient{cc}
}

func (c *resolverServiceClient) Resolve(ctx context.Context, in *ResolveRequest, opts ...grpc.CallOption) (*ResolveResponse, error) {
	out := new(ResolveResponse)
	err := c.cc.Invoke(ctx, "/safing.portmaster.plugin.proto.ResolverService/Resolve", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ResolverServiceServer is the server API for ResolverService service.
// All implementations must embed UnimplementedResolverServiceServer
// for forward compatibility
type ResolverServiceServer interface {
	Resolve(context.Context, *ResolveRequest) (*ResolveResponse, error)
	mustEmbedUnimplementedResolverServiceServer()
}

// UnimplementedResolverServiceServer must be embedded to have forward compatible implementations.
type UnimplementedResolverServiceServer struct {
}

func (UnimplementedResolverServiceServer) Resolve(context.Context, *ResolveRequest) (*ResolveResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Resolve not implemented")
}
func (UnimplementedResolverServiceServer) mustEmbedUnimplementedResolverServiceServer() {}

// UnsafeResolverServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ResolverServiceServer will
// result in compilation errors.
type UnsafeResolverServiceServer interface {
	mustEmbedUnimplementedResolverServiceServer()
}

func RegisterResolverServiceServer(s grpc.ServiceRegistrar, srv ResolverServiceServer) {
	s.RegisterService(&ResolverService_ServiceDesc, srv)
}

func _ResolverService_Resolve_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ResolveRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResolverServiceServer).Resolve(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/safing.portmaster.plugin.proto.ResolverService/Resolve",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResolverServiceServer).Resolve(ctx, req.(*ResolveRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ResolverService_ServiceDesc is the grpc.ServiceDesc for ResolverService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ResolverService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "safing.portmaster.plugin.proto.ResolverService",
	HandlerType: (*ResolverServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Resolve",
			Handler:    _ResolverService_Resolve_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "resolver.proto",
}