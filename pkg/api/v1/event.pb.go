// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.20.0
// source: api/v1/event.proto

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type EventServiceSendRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Events map[string]*MachineProvisioningEvent `protobuf:"bytes,1,rep,name=events,proto3" json:"events,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *EventServiceSendRequest) Reset() {
	*x = EventServiceSendRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_event_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventServiceSendRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventServiceSendRequest) ProtoMessage() {}

func (x *EventServiceSendRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_event_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventServiceSendRequest.ProtoReflect.Descriptor instead.
func (*EventServiceSendRequest) Descriptor() ([]byte, []int) {
	return file_api_v1_event_proto_rawDescGZIP(), []int{0}
}

func (x *EventServiceSendRequest) GetEvents() map[string]*MachineProvisioningEvent {
	if x != nil {
		return x.Events
	}
	return nil
}

type EventServiceSendResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// number of events stored
	Events uint64 `protobuf:"varint,1,opt,name=events,proto3" json:"events,omitempty"`
	// slice of machineIDs for which event was not published
	Failed []string `protobuf:"bytes,2,rep,name=failed,proto3" json:"failed,omitempty"`
}

func (x *EventServiceSendResponse) Reset() {
	*x = EventServiceSendResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_event_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventServiceSendResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventServiceSendResponse) ProtoMessage() {}

func (x *EventServiceSendResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_event_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventServiceSendResponse.ProtoReflect.Descriptor instead.
func (*EventServiceSendResponse) Descriptor() ([]byte, []int) {
	return file_api_v1_event_proto_rawDescGZIP(), []int{1}
}

func (x *EventServiceSendResponse) GetEvents() uint64 {
	if x != nil {
		return x.Events
	}
	return 0
}

func (x *EventServiceSendResponse) GetFailed() []string {
	if x != nil {
		return x.Failed
	}
	return nil
}

type MachineProvisioningEvent struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Time    *timestamppb.Timestamp `protobuf:"bytes,1,opt,name=time,proto3" json:"time,omitempty"`
	Event   string                 `protobuf:"bytes,2,opt,name=event,proto3" json:"event,omitempty"`
	Message string                 `protobuf:"bytes,3,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *MachineProvisioningEvent) Reset() {
	*x = MachineProvisioningEvent{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_event_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MachineProvisioningEvent) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MachineProvisioningEvent) ProtoMessage() {}

func (x *MachineProvisioningEvent) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_event_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MachineProvisioningEvent.ProtoReflect.Descriptor instead.
func (*MachineProvisioningEvent) Descriptor() ([]byte, []int) {
	return file_api_v1_event_proto_rawDescGZIP(), []int{2}
}

func (x *MachineProvisioningEvent) GetTime() *timestamppb.Timestamp {
	if x != nil {
		return x.Time
	}
	return nil
}

func (x *MachineProvisioningEvent) GetEvent() string {
	if x != nil {
		return x.Event
	}
	return ""
}

func (x *MachineProvisioningEvent) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

var File_api_v1_event_proto protoreflect.FileDescriptor

var file_api_v1_event_proto_rawDesc = []byte{
	0x0a, 0x12, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x2f, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x06, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x1a, 0x1f, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xbb, 0x01,
	0x0a, 0x17, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x53, 0x65,
	0x6e, 0x64, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x43, 0x0a, 0x06, 0x65, 0x76, 0x65,
	0x6e, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2b, 0x2e, 0x61, 0x70, 0x69, 0x2e,
	0x76, 0x31, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x53,
	0x65, 0x6e, 0x64, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74,
	0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x1a, 0x5b,
	0x0a, 0x0b, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a,
	0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12,
	0x36, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x20,
	0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x4d, 0x61, 0x63, 0x68, 0x69, 0x6e, 0x65, 0x50,
	0x72, 0x6f, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x69, 0x6e, 0x67, 0x45, 0x76, 0x65, 0x6e, 0x74,
	0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x4a, 0x0a, 0x18, 0x45,
	0x76, 0x65, 0x6e, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x53, 0x65, 0x6e, 0x64, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74,
	0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x12,
	0x16, 0x0a, 0x06, 0x66, 0x61, 0x69, 0x6c, 0x65, 0x64, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52,
	0x06, 0x66, 0x61, 0x69, 0x6c, 0x65, 0x64, 0x22, 0x7a, 0x0a, 0x18, 0x4d, 0x61, 0x63, 0x68, 0x69,
	0x6e, 0x65, 0x50, 0x72, 0x6f, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x69, 0x6e, 0x67, 0x45, 0x76,
	0x65, 0x6e, 0x74, 0x12, 0x2e, 0x0a, 0x04, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x04, 0x74,
	0x69, 0x6d, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x32, 0x5b, 0x0a, 0x0c, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x53, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x12, 0x4b, 0x0a, 0x04, 0x53, 0x65, 0x6e, 0x64, 0x12, 0x1f, 0x2e, 0x61, 0x70,
	0x69, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x53, 0x65, 0x6e, 0x64, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x20, 0x2e, 0x61,
	0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x53, 0x65, 0x6e, 0x64, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00,
	0x42, 0x0a, 0x5a, 0x08, 0x2e, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_api_v1_event_proto_rawDescOnce sync.Once
	file_api_v1_event_proto_rawDescData = file_api_v1_event_proto_rawDesc
)

func file_api_v1_event_proto_rawDescGZIP() []byte {
	file_api_v1_event_proto_rawDescOnce.Do(func() {
		file_api_v1_event_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_v1_event_proto_rawDescData)
	})
	return file_api_v1_event_proto_rawDescData
}

var file_api_v1_event_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_api_v1_event_proto_goTypes = []interface{}{
	(*EventServiceSendRequest)(nil),  // 0: api.v1.EventServiceSendRequest
	(*EventServiceSendResponse)(nil), // 1: api.v1.EventServiceSendResponse
	(*MachineProvisioningEvent)(nil), // 2: api.v1.MachineProvisioningEvent
	nil,                              // 3: api.v1.EventServiceSendRequest.EventsEntry
	(*timestamppb.Timestamp)(nil),    // 4: google.protobuf.Timestamp
}
var file_api_v1_event_proto_depIdxs = []int32{
	3, // 0: api.v1.EventServiceSendRequest.events:type_name -> api.v1.EventServiceSendRequest.EventsEntry
	4, // 1: api.v1.MachineProvisioningEvent.time:type_name -> google.protobuf.Timestamp
	2, // 2: api.v1.EventServiceSendRequest.EventsEntry.value:type_name -> api.v1.MachineProvisioningEvent
	0, // 3: api.v1.EventService.Send:input_type -> api.v1.EventServiceSendRequest
	1, // 4: api.v1.EventService.Send:output_type -> api.v1.EventServiceSendResponse
	4, // [4:5] is the sub-list for method output_type
	3, // [3:4] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_api_v1_event_proto_init() }
func file_api_v1_event_proto_init() {
	if File_api_v1_event_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_api_v1_event_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventServiceSendRequest); i {
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
		file_api_v1_event_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventServiceSendResponse); i {
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
		file_api_v1_event_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MachineProvisioningEvent); i {
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
			RawDescriptor: file_api_v1_event_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_api_v1_event_proto_goTypes,
		DependencyIndexes: file_api_v1_event_proto_depIdxs,
		MessageInfos:      file_api_v1_event_proto_msgTypes,
	}.Build()
	File_api_v1_event_proto = out.File
	file_api_v1_event_proto_rawDesc = nil
	file_api_v1_event_proto_goTypes = nil
	file_api_v1_event_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// EventServiceClient is the client API for EventService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type EventServiceClient interface {
	Send(ctx context.Context, in *EventServiceSendRequest, opts ...grpc.CallOption) (*EventServiceSendResponse, error)
}

type eventServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewEventServiceClient(cc grpc.ClientConnInterface) EventServiceClient {
	return &eventServiceClient{cc}
}

func (c *eventServiceClient) Send(ctx context.Context, in *EventServiceSendRequest, opts ...grpc.CallOption) (*EventServiceSendResponse, error) {
	out := new(EventServiceSendResponse)
	err := c.cc.Invoke(ctx, "/api.v1.EventService/Send", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EventServiceServer is the server API for EventService service.
type EventServiceServer interface {
	Send(context.Context, *EventServiceSendRequest) (*EventServiceSendResponse, error)
}

// UnimplementedEventServiceServer can be embedded to have forward compatible implementations.
type UnimplementedEventServiceServer struct {
}

func (*UnimplementedEventServiceServer) Send(context.Context, *EventServiceSendRequest) (*EventServiceSendResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Send not implemented")
}

func RegisterEventServiceServer(s *grpc.Server, srv EventServiceServer) {
	s.RegisterService(&_EventService_serviceDesc, srv)
}

func _EventService_Send_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EventServiceSendRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EventServiceServer).Send(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.v1.EventService/Send",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EventServiceServer).Send(ctx, req.(*EventServiceSendRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _EventService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "api.v1.EventService",
	HandlerType: (*EventServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Send",
			Handler:    _EventService_Send_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/v1/event.proto",
}
