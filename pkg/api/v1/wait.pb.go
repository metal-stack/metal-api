// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.17.3
// source: api/v1/wait.proto

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type WaitRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	MachineID string `protobuf:"bytes,1,opt,name=machineID,proto3" json:"machineID,omitempty"`
}

func (x *WaitRequest) Reset() {
	*x = WaitRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_wait_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WaitRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WaitRequest) ProtoMessage() {}

func (x *WaitRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_wait_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WaitRequest.ProtoReflect.Descriptor instead.
func (*WaitRequest) Descriptor() ([]byte, []int) {
	return file_api_v1_wait_proto_rawDescGZIP(), []int{0}
}

func (x *WaitRequest) GetMachineID() string {
	if x != nil {
		return x.MachineID
	}
	return ""
}

type KeepPatientResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *KeepPatientResponse) Reset() {
	*x = KeepPatientResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_wait_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *KeepPatientResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*KeepPatientResponse) ProtoMessage() {}

func (x *KeepPatientResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_wait_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use KeepPatientResponse.ProtoReflect.Descriptor instead.
func (*KeepPatientResponse) Descriptor() ([]byte, []int) {
	return file_api_v1_wait_proto_rawDescGZIP(), []int{1}
}

var File_api_v1_wait_proto protoreflect.FileDescriptor

var file_api_v1_wait_proto_rawDesc = []byte{
	0x0a, 0x11, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x2f, 0x77, 0x61, 0x69, 0x74, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x02, 0x76, 0x31, 0x22, 0x2b, 0x0a, 0x0b, 0x57, 0x61, 0x69, 0x74, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x6d, 0x61, 0x63, 0x68, 0x69, 0x6e,
	0x65, 0x49, 0x44, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6d, 0x61, 0x63, 0x68, 0x69,
	0x6e, 0x65, 0x49, 0x44, 0x22, 0x15, 0x0a, 0x13, 0x4b, 0x65, 0x65, 0x70, 0x50, 0x61, 0x74, 0x69,
	0x65, 0x6e, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x32, 0x3a, 0x0a, 0x04, 0x57,
	0x61, 0x69, 0x74, 0x12, 0x32, 0x0a, 0x04, 0x57, 0x61, 0x69, 0x74, 0x12, 0x0f, 0x2e, 0x76, 0x31,
	0x2e, 0x57, 0x61, 0x69, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x17, 0x2e, 0x76,
	0x31, 0x2e, 0x4b, 0x65, 0x65, 0x70, 0x50, 0x61, 0x74, 0x69, 0x65, 0x6e, 0x74, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x30, 0x01, 0x42, 0x08, 0x5a, 0x06, 0x61, 0x70, 0x69, 0x2f, 0x76,
	0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_api_v1_wait_proto_rawDescOnce sync.Once
	file_api_v1_wait_proto_rawDescData = file_api_v1_wait_proto_rawDesc
)

func file_api_v1_wait_proto_rawDescGZIP() []byte {
	file_api_v1_wait_proto_rawDescOnce.Do(func() {
		file_api_v1_wait_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_v1_wait_proto_rawDescData)
	})
	return file_api_v1_wait_proto_rawDescData
}

var file_api_v1_wait_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_api_v1_wait_proto_goTypes = []interface{}{
	(*WaitRequest)(nil),         // 0: v1.WaitRequest
	(*KeepPatientResponse)(nil), // 1: v1.KeepPatientResponse
}
var file_api_v1_wait_proto_depIdxs = []int32{
	0, // 0: v1.Wait.Wait:input_type -> v1.WaitRequest
	1, // 1: v1.Wait.Wait:output_type -> v1.KeepPatientResponse
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_api_v1_wait_proto_init() }
func file_api_v1_wait_proto_init() {
	if File_api_v1_wait_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_api_v1_wait_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WaitRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_v1_wait_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*KeepPatientResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_api_v1_wait_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_api_v1_wait_proto_goTypes,
		DependencyIndexes: file_api_v1_wait_proto_depIdxs,
		MessageInfos:      file_api_v1_wait_proto_msgTypes,
	}.Build()
	File_api_v1_wait_proto = out.File
	file_api_v1_wait_proto_rawDesc = nil
	file_api_v1_wait_proto_goTypes = nil
	file_api_v1_wait_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// WaitClient is the client API for Wait service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type WaitClient interface {
	Wait(ctx context.Context, in *WaitRequest, opts ...grpc.CallOption) (Wait_WaitClient, error)
}

type waitClient struct {
	cc grpc.ClientConnInterface
}

func NewWaitClient(cc grpc.ClientConnInterface) WaitClient {
	return &waitClient{cc}
}

func (c *waitClient) Wait(ctx context.Context, in *WaitRequest, opts ...grpc.CallOption) (Wait_WaitClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Wait_serviceDesc.Streams[0], "/v1.Wait/Wait", opts...)
	if err != nil {
		return nil, err
	}
	x := &waitWaitClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Wait_WaitClient interface {
	Recv() (*KeepPatientResponse, error)
	grpc.ClientStream
}

type waitWaitClient struct {
	grpc.ClientStream
}

func (x *waitWaitClient) Recv() (*KeepPatientResponse, error) {
	m := new(KeepPatientResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// WaitServer is the server API for Wait service.
type WaitServer interface {
	Wait(*WaitRequest, Wait_WaitServer) error
}

// UnimplementedWaitServer can be embedded to have forward compatible implementations.
type UnimplementedWaitServer struct {
}

func (*UnimplementedWaitServer) Wait(*WaitRequest, Wait_WaitServer) error {
	return status.Errorf(codes.Unimplemented, "method Wait not implemented")
}

func RegisterWaitServer(s *grpc.Server, srv WaitServer) {
	s.RegisterService(&_Wait_serviceDesc, srv)
}

func _Wait_Wait_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(WaitRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(WaitServer).Wait(m, &waitWaitServer{stream})
}

type Wait_WaitServer interface {
	Send(*KeepPatientResponse) error
	grpc.ServerStream
}

type waitWaitServer struct {
	grpc.ServerStream
}

func (x *waitWaitServer) Send(m *KeepPatientResponse) error {
	return x.ServerStream.SendMsg(m)
}

var _Wait_serviceDesc = grpc.ServiceDesc{
	ServiceName: "v1.Wait",
	HandlerType: (*WaitServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Wait",
			Handler:       _Wait_Wait_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "api/v1/wait.proto",
}
