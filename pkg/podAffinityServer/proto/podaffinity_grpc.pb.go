// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v3.21.11
// source: podaffinity.proto

package service

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	PodAffinityAware_GetNumaNodeInof_FullMethodName = "/PodAffinityAware/GetNumaNodeInof"
	PodAffinityAware_GetPodAffinity_FullMethodName  = "/PodAffinityAware/GetPodAffinity"
)

// PodAffinityAwareClient is the client API for PodAffinityAware service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type PodAffinityAwareClient interface {
	GetNumaNodeInof(ctx context.Context, in *NullRequest, opts ...grpc.CallOption) (*NumaInfoResponse, error)
	GetPodAffinity(ctx context.Context, in *AffinityRequest, opts ...grpc.CallOption) (*AffinityResponse, error)
}

type podAffinityAwareClient struct {
	cc grpc.ClientConnInterface
}

func NewPodAffinityAwareClient(cc grpc.ClientConnInterface) PodAffinityAwareClient {
	return &podAffinityAwareClient{cc}
}

func (c *podAffinityAwareClient) GetNumaNodeInof(ctx context.Context, in *NullRequest, opts ...grpc.CallOption) (*NumaInfoResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(NumaInfoResponse)
	err := c.cc.Invoke(ctx, PodAffinityAware_GetNumaNodeInof_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *podAffinityAwareClient) GetPodAffinity(ctx context.Context, in *AffinityRequest, opts ...grpc.CallOption) (*AffinityResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(AffinityResponse)
	err := c.cc.Invoke(ctx, PodAffinityAware_GetPodAffinity_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PodAffinityAwareServer is the server API for PodAffinityAware service.
// All implementations must embed UnimplementedPodAffinityAwareServer
// for forward compatibility.
type PodAffinityAwareServer interface {
	GetNumaNodeInof(context.Context, *NullRequest) (*NumaInfoResponse, error)
	GetPodAffinity(context.Context, *AffinityRequest) (*AffinityResponse, error)
	mustEmbedUnimplementedPodAffinityAwareServer()
}

// UnimplementedPodAffinityAwareServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedPodAffinityAwareServer struct{}

func (UnimplementedPodAffinityAwareServer) GetNumaNodeInof(context.Context, *NullRequest) (*NumaInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetNumaNodeInof not implemented")
}
func (UnimplementedPodAffinityAwareServer) GetPodAffinity(context.Context, *AffinityRequest) (*AffinityResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPodAffinity not implemented")
}
func (UnimplementedPodAffinityAwareServer) mustEmbedUnimplementedPodAffinityAwareServer() {}
func (UnimplementedPodAffinityAwareServer) testEmbeddedByValue()                          {}

// UnsafePodAffinityAwareServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to PodAffinityAwareServer will
// result in compilation errors.
type UnsafePodAffinityAwareServer interface {
	mustEmbedUnimplementedPodAffinityAwareServer()
}

func RegisterPodAffinityAwareServer(s grpc.ServiceRegistrar, srv PodAffinityAwareServer) {
	// If the following call pancis, it indicates UnimplementedPodAffinityAwareServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&PodAffinityAware_ServiceDesc, srv)
}

func _PodAffinityAware_GetNumaNodeInof_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NullRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PodAffinityAwareServer).GetNumaNodeInof(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PodAffinityAware_GetNumaNodeInof_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PodAffinityAwareServer).GetNumaNodeInof(ctx, req.(*NullRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PodAffinityAware_GetPodAffinity_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AffinityRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PodAffinityAwareServer).GetPodAffinity(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PodAffinityAware_GetPodAffinity_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PodAffinityAwareServer).GetPodAffinity(ctx, req.(*AffinityRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// PodAffinityAware_ServiceDesc is the grpc.ServiceDesc for PodAffinityAware service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var PodAffinityAware_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "PodAffinityAware",
	HandlerType: (*PodAffinityAwareServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetNumaNodeInof",
			Handler:    _PodAffinityAware_GetNumaNodeInof_Handler,
		},
		{
			MethodName: "GetPodAffinity",
			Handler:    _PodAffinityAware_GetPodAffinity_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "podaffinity.proto",
}
