// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: api/v1/boot.proto

package v1

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

const (
	BootService_Dhcp_FullMethodName              = "/api.v1.BootService/Dhcp"
	BootService_Boot_FullMethodName              = "/api.v1.BootService/Boot"
	BootService_SuperUserPassword_FullMethodName = "/api.v1.BootService/SuperUserPassword"
	BootService_Register_FullMethodName          = "/api.v1.BootService/Register"
	BootService_Wait_FullMethodName              = "/api.v1.BootService/Wait"
	BootService_Report_FullMethodName            = "/api.v1.BootService/Report"
	BootService_AbortReinstall_FullMethodName    = "/api.v1.BootService/AbortReinstall"
)

// BootServiceClient is the client API for BootService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BootServiceClient interface {
	// Dhcp is the first dhcp request (option 97). A ProvisioningEventPXEBooting is fired
	Dhcp(ctx context.Context, in *BootServiceDhcpRequest, opts ...grpc.CallOption) (*BootServiceDhcpResponse, error)
	// Boot is called from pixie once the machine got the first dhcp response and ipxie asks for subsequent kernel and initrd
	Boot(ctx context.Context, in *BootServiceBootRequest, opts ...grpc.CallOption) (*BootServiceBootResponse, error)
	// SuperUserPassword metal-hammer takes the configured root password for the bmc from metal-api and configure the bmc accordingly
	SuperUserPassword(ctx context.Context, in *BootServiceSuperUserPasswordRequest, opts ...grpc.CallOption) (*BootServiceSuperUserPasswordResponse, error)
	// Register is called from metal-hammer after hardware inventory is finished, tells metal-api all glory details about that machine
	Register(ctx context.Context, in *BootServiceRegisterRequest, opts ...grpc.CallOption) (*BootServiceRegisterResponse, error)
	// Wait is a hanging call that waits until the machine gets allocated by a user
	Wait(ctx context.Context, in *BootServiceWaitRequest, opts ...grpc.CallOption) (BootService_WaitClient, error)
	// Report tells metal-api installation was either successful or failed
	Report(ctx context.Context, in *BootServiceReportRequest, opts ...grpc.CallOption) (*BootServiceReportResponse, error)
	// If reinstall failed and tell metal-api to restore to previous state
	AbortReinstall(ctx context.Context, in *BootServiceAbortReinstallRequest, opts ...grpc.CallOption) (*BootServiceAbortReinstallResponse, error)
}

type bootServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewBootServiceClient(cc grpc.ClientConnInterface) BootServiceClient {
	return &bootServiceClient{cc}
}

func (c *bootServiceClient) Dhcp(ctx context.Context, in *BootServiceDhcpRequest, opts ...grpc.CallOption) (*BootServiceDhcpResponse, error) {
	out := new(BootServiceDhcpResponse)
	err := c.cc.Invoke(ctx, BootService_Dhcp_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bootServiceClient) Boot(ctx context.Context, in *BootServiceBootRequest, opts ...grpc.CallOption) (*BootServiceBootResponse, error) {
	out := new(BootServiceBootResponse)
	err := c.cc.Invoke(ctx, BootService_Boot_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bootServiceClient) SuperUserPassword(ctx context.Context, in *BootServiceSuperUserPasswordRequest, opts ...grpc.CallOption) (*BootServiceSuperUserPasswordResponse, error) {
	out := new(BootServiceSuperUserPasswordResponse)
	err := c.cc.Invoke(ctx, BootService_SuperUserPassword_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bootServiceClient) Register(ctx context.Context, in *BootServiceRegisterRequest, opts ...grpc.CallOption) (*BootServiceRegisterResponse, error) {
	out := new(BootServiceRegisterResponse)
	err := c.cc.Invoke(ctx, BootService_Register_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bootServiceClient) Wait(ctx context.Context, in *BootServiceWaitRequest, opts ...grpc.CallOption) (BootService_WaitClient, error) {
	stream, err := c.cc.NewStream(ctx, &BootService_ServiceDesc.Streams[0], BootService_Wait_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &bootServiceWaitClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type BootService_WaitClient interface {
	Recv() (*BootServiceWaitResponse, error)
	grpc.ClientStream
}

type bootServiceWaitClient struct {
	grpc.ClientStream
}

func (x *bootServiceWaitClient) Recv() (*BootServiceWaitResponse, error) {
	m := new(BootServiceWaitResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *bootServiceClient) Report(ctx context.Context, in *BootServiceReportRequest, opts ...grpc.CallOption) (*BootServiceReportResponse, error) {
	out := new(BootServiceReportResponse)
	err := c.cc.Invoke(ctx, BootService_Report_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bootServiceClient) AbortReinstall(ctx context.Context, in *BootServiceAbortReinstallRequest, opts ...grpc.CallOption) (*BootServiceAbortReinstallResponse, error) {
	out := new(BootServiceAbortReinstallResponse)
	err := c.cc.Invoke(ctx, BootService_AbortReinstall_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BootServiceServer is the server API for BootService service.
// All implementations should embed UnimplementedBootServiceServer
// for forward compatibility
type BootServiceServer interface {
	// Dhcp is the first dhcp request (option 97). A ProvisioningEventPXEBooting is fired
	Dhcp(context.Context, *BootServiceDhcpRequest) (*BootServiceDhcpResponse, error)
	// Boot is called from pixie once the machine got the first dhcp response and ipxie asks for subsequent kernel and initrd
	Boot(context.Context, *BootServiceBootRequest) (*BootServiceBootResponse, error)
	// SuperUserPassword metal-hammer takes the configured root password for the bmc from metal-api and configure the bmc accordingly
	SuperUserPassword(context.Context, *BootServiceSuperUserPasswordRequest) (*BootServiceSuperUserPasswordResponse, error)
	// Register is called from metal-hammer after hardware inventory is finished, tells metal-api all glory details about that machine
	Register(context.Context, *BootServiceRegisterRequest) (*BootServiceRegisterResponse, error)
	// Wait is a hanging call that waits until the machine gets allocated by a user
	Wait(*BootServiceWaitRequest, BootService_WaitServer) error
	// Report tells metal-api installation was either successful or failed
	Report(context.Context, *BootServiceReportRequest) (*BootServiceReportResponse, error)
	// If reinstall failed and tell metal-api to restore to previous state
	AbortReinstall(context.Context, *BootServiceAbortReinstallRequest) (*BootServiceAbortReinstallResponse, error)
}

// UnimplementedBootServiceServer should be embedded to have forward compatible implementations.
type UnimplementedBootServiceServer struct {
}

func (UnimplementedBootServiceServer) Dhcp(context.Context, *BootServiceDhcpRequest) (*BootServiceDhcpResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Dhcp not implemented")
}
func (UnimplementedBootServiceServer) Boot(context.Context, *BootServiceBootRequest) (*BootServiceBootResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Boot not implemented")
}
func (UnimplementedBootServiceServer) SuperUserPassword(context.Context, *BootServiceSuperUserPasswordRequest) (*BootServiceSuperUserPasswordResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SuperUserPassword not implemented")
}
func (UnimplementedBootServiceServer) Register(context.Context, *BootServiceRegisterRequest) (*BootServiceRegisterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Register not implemented")
}
func (UnimplementedBootServiceServer) Wait(*BootServiceWaitRequest, BootService_WaitServer) error {
	return status.Errorf(codes.Unimplemented, "method Wait not implemented")
}
func (UnimplementedBootServiceServer) Report(context.Context, *BootServiceReportRequest) (*BootServiceReportResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Report not implemented")
}
func (UnimplementedBootServiceServer) AbortReinstall(context.Context, *BootServiceAbortReinstallRequest) (*BootServiceAbortReinstallResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AbortReinstall not implemented")
}

// UnsafeBootServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BootServiceServer will
// result in compilation errors.
type UnsafeBootServiceServer interface {
	mustEmbedUnimplementedBootServiceServer()
}

func RegisterBootServiceServer(s grpc.ServiceRegistrar, srv BootServiceServer) {
	s.RegisterService(&BootService_ServiceDesc, srv)
}

func _BootService_Dhcp_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BootServiceDhcpRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BootServiceServer).Dhcp(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BootService_Dhcp_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BootServiceServer).Dhcp(ctx, req.(*BootServiceDhcpRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BootService_Boot_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BootServiceBootRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BootServiceServer).Boot(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BootService_Boot_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BootServiceServer).Boot(ctx, req.(*BootServiceBootRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BootService_SuperUserPassword_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BootServiceSuperUserPasswordRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BootServiceServer).SuperUserPassword(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BootService_SuperUserPassword_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BootServiceServer).SuperUserPassword(ctx, req.(*BootServiceSuperUserPasswordRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BootService_Register_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BootServiceRegisterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BootServiceServer).Register(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BootService_Register_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BootServiceServer).Register(ctx, req.(*BootServiceRegisterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BootService_Wait_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(BootServiceWaitRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(BootServiceServer).Wait(m, &bootServiceWaitServer{stream})
}

type BootService_WaitServer interface {
	Send(*BootServiceWaitResponse) error
	grpc.ServerStream
}

type bootServiceWaitServer struct {
	grpc.ServerStream
}

func (x *bootServiceWaitServer) Send(m *BootServiceWaitResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _BootService_Report_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BootServiceReportRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BootServiceServer).Report(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BootService_Report_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BootServiceServer).Report(ctx, req.(*BootServiceReportRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BootService_AbortReinstall_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BootServiceAbortReinstallRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BootServiceServer).AbortReinstall(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BootService_AbortReinstall_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BootServiceServer).AbortReinstall(ctx, req.(*BootServiceAbortReinstallRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// BootService_ServiceDesc is the grpc.ServiceDesc for BootService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BootService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "api.v1.BootService",
	HandlerType: (*BootServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Dhcp",
			Handler:    _BootService_Dhcp_Handler,
		},
		{
			MethodName: "Boot",
			Handler:    _BootService_Boot_Handler,
		},
		{
			MethodName: "SuperUserPassword",
			Handler:    _BootService_SuperUserPassword_Handler,
		},
		{
			MethodName: "Register",
			Handler:    _BootService_Register_Handler,
		},
		{
			MethodName: "Report",
			Handler:    _BootService_Report_Handler,
		},
		{
			MethodName: "AbortReinstall",
			Handler:    _BootService_AbortReinstall_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Wait",
			Handler:       _BootService_Wait_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "api/v1/boot.proto",
}
