// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.2
// source: decider.proto

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

// DeciderServiceClient is the client API for DeciderService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DeciderServiceClient interface {
	DecideOnConnection(ctx context.Context, in *DecideOnConnectionRequest, opts ...grpc.CallOption) (*DecideOnConnectionResponse, error)
}

type deciderServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewDeciderServiceClient(cc grpc.ClientConnInterface) DeciderServiceClient {
	return &deciderServiceClient{cc}
}

func (c *deciderServiceClient) DecideOnConnection(ctx context.Context, in *DecideOnConnectionRequest, opts ...grpc.CallOption) (*DecideOnConnectionResponse, error) {
	out := new(DecideOnConnectionResponse)
	err := c.cc.Invoke(ctx, "/safing.portmaster.plugin.proto.DeciderService/DecideOnConnection", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DeciderServiceServer is the server API for DeciderService service.
// All implementations must embed UnimplementedDeciderServiceServer
// for forward compatibility
type DeciderServiceServer interface {
	DecideOnConnection(context.Context, *DecideOnConnectionRequest) (*DecideOnConnectionResponse, error)
	mustEmbedUnimplementedDeciderServiceServer()
}

// UnimplementedDeciderServiceServer must be embedded to have forward compatible implementations.
type UnimplementedDeciderServiceServer struct {
}

func (UnimplementedDeciderServiceServer) DecideOnConnection(context.Context, *DecideOnConnectionRequest) (*DecideOnConnectionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DecideOnConnection not implemented")
}
func (UnimplementedDeciderServiceServer) mustEmbedUnimplementedDeciderServiceServer() {}

// UnsafeDeciderServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DeciderServiceServer will
// result in compilation errors.
type UnsafeDeciderServiceServer interface {
	mustEmbedUnimplementedDeciderServiceServer()
}

func RegisterDeciderServiceServer(s grpc.ServiceRegistrar, srv DeciderServiceServer) {
	s.RegisterService(&DeciderService_ServiceDesc, srv)
}

func _DeciderService_DecideOnConnection_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DecideOnConnectionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeciderServiceServer).DecideOnConnection(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/safing.portmaster.plugin.proto.DeciderService/DecideOnConnection",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeciderServiceServer).DecideOnConnection(ctx, req.(*DecideOnConnectionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// DeciderService_ServiceDesc is the grpc.ServiceDesc for DeciderService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DeciderService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "safing.portmaster.plugin.proto.DeciderService",
	HandlerType: (*DeciderServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "DecideOnConnection",
			Handler:    _DeciderService_DecideOnConnection_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "decider.proto",
}
