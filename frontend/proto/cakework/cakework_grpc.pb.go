// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.20.3
// source: proto/cakework/cakework.proto

package cakework

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

// CakeworkClient is the client API for Cakework service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type CakeworkClient interface {
	RunActivity(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Reply, error)
}

type cakeworkClient struct {
	cc grpc.ClientConnInterface
}

func NewCakeworkClient(cc grpc.ClientConnInterface) CakeworkClient {
	return &cakeworkClient{cc}
}

func (c *cakeworkClient) RunActivity(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Reply, error) {
	out := new(Reply)
	err := c.cc.Invoke(ctx, "/Cakework/RunActivity", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CakeworkServer is the server API for Cakework service.
// All implementations must embed UnimplementedCakeworkServer
// for forward compatibility
type CakeworkServer interface {
	RunActivity(context.Context, *Request) (*Reply, error)
	mustEmbedUnimplementedCakeworkServer()
}

// UnimplementedCakeworkServer must be embedded to have forward compatible implementations.
type UnimplementedCakeworkServer struct {
}

func (UnimplementedCakeworkServer) RunActivity(context.Context, *Request) (*Reply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RunActivity not implemented")
}
func (UnimplementedCakeworkServer) mustEmbedUnimplementedCakeworkServer() {}

// UnsafeCakeworkServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to CakeworkServer will
// result in compilation errors.
type UnsafeCakeworkServer interface {
	mustEmbedUnimplementedCakeworkServer()
}

func RegisterCakeworkServer(s grpc.ServiceRegistrar, srv CakeworkServer) {
	s.RegisterService(&Cakework_ServiceDesc, srv)
}

func _Cakework_RunActivity_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CakeworkServer).RunActivity(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Cakework/RunActivity",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CakeworkServer).RunActivity(ctx, req.(*Request))
	}
	return interceptor(ctx, in, info, handler)
}

// Cakework_ServiceDesc is the grpc.ServiceDesc for Cakework service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Cakework_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "Cakework",
	HandlerType: (*CakeworkServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RunActivity",
			Handler:    _Cakework_RunActivity_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "cakework.proto",
}
