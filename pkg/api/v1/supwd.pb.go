// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.20.0
// source: api/v1/supwd.proto

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

type SuperUserPasswordRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *SuperUserPasswordRequest) Reset() {
	*x = SuperUserPasswordRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_supwd_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SuperUserPasswordRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SuperUserPasswordRequest) ProtoMessage() {}

func (x *SuperUserPasswordRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_supwd_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SuperUserPasswordRequest.ProtoReflect.Descriptor instead.
func (*SuperUserPasswordRequest) Descriptor() ([]byte, []int) {
	return file_api_v1_supwd_proto_rawDescGZIP(), []int{0}
}

type SuperUserPasswordResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	FeatureDisabled   bool   `protobuf:"varint,1,opt,name=featureDisabled,proto3" json:"featureDisabled,omitempty"`
	SuperUserPassword string `protobuf:"bytes,2,opt,name=superUserPassword,proto3" json:"superUserPassword,omitempty"`
}

func (x *SuperUserPasswordResponse) Reset() {
	*x = SuperUserPasswordResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_supwd_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SuperUserPasswordResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SuperUserPasswordResponse) ProtoMessage() {}

func (x *SuperUserPasswordResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_supwd_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SuperUserPasswordResponse.ProtoReflect.Descriptor instead.
func (*SuperUserPasswordResponse) Descriptor() ([]byte, []int) {
	return file_api_v1_supwd_proto_rawDescGZIP(), []int{1}
}

func (x *SuperUserPasswordResponse) GetFeatureDisabled() bool {
	if x != nil {
		return x.FeatureDisabled
	}
	return false
}

func (x *SuperUserPasswordResponse) GetSuperUserPassword() string {
	if x != nil {
		return x.SuperUserPassword
	}
	return ""
}

var File_api_v1_supwd_proto protoreflect.FileDescriptor

var file_api_v1_supwd_proto_rawDesc = []byte{
	0x0a, 0x12, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x2f, 0x73, 0x75, 0x70, 0x77, 0x64, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x02, 0x76, 0x31, 0x22, 0x1a, 0x0a, 0x18, 0x53, 0x75, 0x70, 0x65,
	0x72, 0x55, 0x73, 0x65, 0x72, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x22, 0x73, 0x0a, 0x19, 0x53, 0x75, 0x70, 0x65, 0x72, 0x55, 0x73, 0x65,
	0x72, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x28, 0x0a, 0x0f, 0x66, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x44, 0x69, 0x73, 0x61,
	0x62, 0x6c, 0x65, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0f, 0x66, 0x65, 0x61, 0x74,
	0x75, 0x72, 0x65, 0x44, 0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x64, 0x12, 0x2c, 0x0a, 0x11, 0x73,
	0x75, 0x70, 0x65, 0x72, 0x55, 0x73, 0x65, 0x72, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x11, 0x73, 0x75, 0x70, 0x65, 0x72, 0x55, 0x73, 0x65,
	0x72, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x32, 0x6a, 0x0a, 0x11, 0x53, 0x75, 0x70,
	0x65, 0x72, 0x55, 0x73, 0x65, 0x72, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x12, 0x55,
	0x0a, 0x16, 0x46, 0x65, 0x74, 0x63, 0x68, 0x53, 0x75, 0x70, 0x65, 0x72, 0x55, 0x73, 0x65, 0x72,
	0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x12, 0x1c, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x75,
	0x70, 0x65, 0x72, 0x55, 0x73, 0x65, 0x72, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1d, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x75, 0x70, 0x65,
	0x72, 0x55, 0x73, 0x65, 0x72, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x08, 0x5a, 0x06, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_api_v1_supwd_proto_rawDescOnce sync.Once
	file_api_v1_supwd_proto_rawDescData = file_api_v1_supwd_proto_rawDesc
)

func file_api_v1_supwd_proto_rawDescGZIP() []byte {
	file_api_v1_supwd_proto_rawDescOnce.Do(func() {
		file_api_v1_supwd_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_v1_supwd_proto_rawDescData)
	})
	return file_api_v1_supwd_proto_rawDescData
}

var file_api_v1_supwd_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_api_v1_supwd_proto_goTypes = []interface{}{
	(*SuperUserPasswordRequest)(nil),  // 0: v1.SuperUserPasswordRequest
	(*SuperUserPasswordResponse)(nil), // 1: v1.SuperUserPasswordResponse
}
var file_api_v1_supwd_proto_depIdxs = []int32{
	0, // 0: v1.SuperUserPassword.FetchSuperUserPassword:input_type -> v1.SuperUserPasswordRequest
	1, // 1: v1.SuperUserPassword.FetchSuperUserPassword:output_type -> v1.SuperUserPasswordResponse
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_api_v1_supwd_proto_init() }
func file_api_v1_supwd_proto_init() {
	if File_api_v1_supwd_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_api_v1_supwd_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SuperUserPasswordRequest); i {
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
		file_api_v1_supwd_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SuperUserPasswordResponse); i {
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
			RawDescriptor: file_api_v1_supwd_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_api_v1_supwd_proto_goTypes,
		DependencyIndexes: file_api_v1_supwd_proto_depIdxs,
		MessageInfos:      file_api_v1_supwd_proto_msgTypes,
	}.Build()
	File_api_v1_supwd_proto = out.File
	file_api_v1_supwd_proto_rawDesc = nil
	file_api_v1_supwd_proto_goTypes = nil
	file_api_v1_supwd_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// SuperUserPasswordClient is the client API for SuperUserPassword service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type SuperUserPasswordClient interface {
	FetchSuperUserPassword(ctx context.Context, in *SuperUserPasswordRequest, opts ...grpc.CallOption) (*SuperUserPasswordResponse, error)
}

type superUserPasswordClient struct {
	cc grpc.ClientConnInterface
}

func NewSuperUserPasswordClient(cc grpc.ClientConnInterface) SuperUserPasswordClient {
	return &superUserPasswordClient{cc}
}

func (c *superUserPasswordClient) FetchSuperUserPassword(ctx context.Context, in *SuperUserPasswordRequest, opts ...grpc.CallOption) (*SuperUserPasswordResponse, error) {
	out := new(SuperUserPasswordResponse)
	err := c.cc.Invoke(ctx, "/v1.SuperUserPassword/FetchSuperUserPassword", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SuperUserPasswordServer is the server API for SuperUserPassword service.
type SuperUserPasswordServer interface {
	FetchSuperUserPassword(context.Context, *SuperUserPasswordRequest) (*SuperUserPasswordResponse, error)
}

// UnimplementedSuperUserPasswordServer can be embedded to have forward compatible implementations.
type UnimplementedSuperUserPasswordServer struct {
}

func (*UnimplementedSuperUserPasswordServer) FetchSuperUserPassword(context.Context, *SuperUserPasswordRequest) (*SuperUserPasswordResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FetchSuperUserPassword not implemented")
}

func RegisterSuperUserPasswordServer(s *grpc.Server, srv SuperUserPasswordServer) {
	s.RegisterService(&_SuperUserPassword_serviceDesc, srv)
}

func _SuperUserPassword_FetchSuperUserPassword_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SuperUserPasswordRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SuperUserPasswordServer).FetchSuperUserPassword(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.SuperUserPassword/FetchSuperUserPassword",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SuperUserPasswordServer).FetchSuperUserPassword(ctx, req.(*SuperUserPasswordRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _SuperUserPassword_serviceDesc = grpc.ServiceDesc{
	ServiceName: "v1.SuperUserPassword",
	HandlerType: (*SuperUserPasswordServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "FetchSuperUserPassword",
			Handler:    _SuperUserPassword_FetchSuperUserPassword_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/v1/supwd.proto",
}
