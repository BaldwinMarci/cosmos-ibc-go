// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: ibc/applications/fee/v1/tx.proto

package types

import (
	context "context"
	fmt "fmt"
	types "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	_ "github.com/gogo/protobuf/gogoproto"
	grpc1 "github.com/gogo/protobuf/grpc"
	proto "github.com/gogo/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// MsgRegisterCounterpartyAddress is the request type for registering the counter party address
type MsgRegisterCounterpartyAddress struct {
	Address             string `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	CounterpartyAddress string `protobuf:"bytes,2,opt,name=counterparty_address,json=counterpartyAddress,proto3" json:"counterparty_address,omitempty" yaml:"counterparty_address"`
}

func (m *MsgRegisterCounterpartyAddress) Reset()         { *m = MsgRegisterCounterpartyAddress{} }
func (m *MsgRegisterCounterpartyAddress) String() string { return proto.CompactTextString(m) }
func (*MsgRegisterCounterpartyAddress) ProtoMessage()    {}
func (*MsgRegisterCounterpartyAddress) Descriptor() ([]byte, []int) {
	return fileDescriptor_05c93128649f1b96, []int{0}
}
func (m *MsgRegisterCounterpartyAddress) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgRegisterCounterpartyAddress) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgRegisterCounterpartyAddress.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgRegisterCounterpartyAddress) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgRegisterCounterpartyAddress.Merge(m, src)
}
func (m *MsgRegisterCounterpartyAddress) XXX_Size() int {
	return m.Size()
}
func (m *MsgRegisterCounterpartyAddress) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgRegisterCounterpartyAddress.DiscardUnknown(m)
}

var xxx_messageInfo_MsgRegisterCounterpartyAddress proto.InternalMessageInfo

// MsgRegisterCounterPartyAddressResponse defines the Msg/RegisterCounteryPartyAddress response type
type MsgRegisterCounterPartyAddressResponse struct {
}

func (m *MsgRegisterCounterPartyAddressResponse) Reset() {
	*m = MsgRegisterCounterPartyAddressResponse{}
}
func (m *MsgRegisterCounterPartyAddressResponse) String() string { return proto.CompactTextString(m) }
func (*MsgRegisterCounterPartyAddressResponse) ProtoMessage()    {}
func (*MsgRegisterCounterPartyAddressResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_05c93128649f1b96, []int{1}
}
func (m *MsgRegisterCounterPartyAddressResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgRegisterCounterPartyAddressResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgRegisterCounterPartyAddressResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgRegisterCounterPartyAddressResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgRegisterCounterPartyAddressResponse.Merge(m, src)
}
func (m *MsgRegisterCounterPartyAddressResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgRegisterCounterPartyAddressResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgRegisterCounterPartyAddressResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgRegisterCounterPartyAddressResponse proto.InternalMessageInfo

// MsgEscrowPacketFee defines the request type EscrowPacketFee RPC
type MsgEscrowPacketFee struct {
	PacketId   *types.PacketId `protobuf:"bytes,1,opt,name=packet_id,json=packetId,proto3" json:"packet_id,omitempty" yaml:"packet_id"`
	ReceiveFee *Fee            `protobuf:"bytes,2,opt,name=receive_fee,json=receiveFee,proto3" json:"receive_fee,omitempty" yaml:"receive_fee"`
	AckFee     *Fee            `protobuf:"bytes,3,opt,name=ack_fee,json=ackFee,proto3" json:"ack_fee,omitempty" yaml:"ack_fee"`
	TimeoutFee *Fee            `protobuf:"bytes,4,opt,name=timeout_fee,json=timeoutFee,proto3" json:"timeout_fee,omitempty" yaml:"timeout_fee"`
	Relayers   []string        `protobuf:"bytes,5,rep,name=relayers,proto3" json:"relayers,omitempty"`
}

func (m *MsgEscrowPacketFee) Reset()         { *m = MsgEscrowPacketFee{} }
func (m *MsgEscrowPacketFee) String() string { return proto.CompactTextString(m) }
func (*MsgEscrowPacketFee) ProtoMessage()    {}
func (*MsgEscrowPacketFee) Descriptor() ([]byte, []int) {
	return fileDescriptor_05c93128649f1b96, []int{2}
}
func (m *MsgEscrowPacketFee) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgEscrowPacketFee) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgEscrowPacketFee.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgEscrowPacketFee) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgEscrowPacketFee.Merge(m, src)
}
func (m *MsgEscrowPacketFee) XXX_Size() int {
	return m.Size()
}
func (m *MsgEscrowPacketFee) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgEscrowPacketFee.DiscardUnknown(m)
}

var xxx_messageInfo_MsgEscrowPacketFee proto.InternalMessageInfo

// MsgEscrowPacketFeeResponse defines the response type for Msg/EscrowPacketFee
type MsgEscrowPacketFeeResponse struct {
}

func (m *MsgEscrowPacketFeeResponse) Reset()         { *m = MsgEscrowPacketFeeResponse{} }
func (m *MsgEscrowPacketFeeResponse) String() string { return proto.CompactTextString(m) }
func (*MsgEscrowPacketFeeResponse) ProtoMessage()    {}
func (*MsgEscrowPacketFeeResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_05c93128649f1b96, []int{3}
}
func (m *MsgEscrowPacketFeeResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgEscrowPacketFeeResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgEscrowPacketFeeResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgEscrowPacketFeeResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgEscrowPacketFeeResponse.Merge(m, src)
}
func (m *MsgEscrowPacketFeeResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgEscrowPacketFeeResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgEscrowPacketFeeResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgEscrowPacketFeeResponse proto.InternalMessageInfo

func init() {
	proto.RegisterType((*MsgRegisterCounterpartyAddress)(nil), "ibc.applications.fee.v1.MsgRegisterCounterpartyAddress")
	proto.RegisterType((*MsgRegisterCounterPartyAddressResponse)(nil), "ibc.applications.fee.v1.MsgRegisterCounterPartyAddressResponse")
	proto.RegisterType((*MsgEscrowPacketFee)(nil), "ibc.applications.fee.v1.MsgEscrowPacketFee")
	proto.RegisterType((*MsgEscrowPacketFeeResponse)(nil), "ibc.applications.fee.v1.MsgEscrowPacketFeeResponse")
}

func init() { proto.RegisterFile("ibc/applications/fee/v1/tx.proto", fileDescriptor_05c93128649f1b96) }

var fileDescriptor_05c93128649f1b96 = []byte{
	// 531 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x93, 0x4f, 0x6b, 0x13, 0x41,
	0x18, 0xc6, 0xb3, 0x8d, 0xb6, 0xc9, 0x04, 0x54, 0xc6, 0xa0, 0x61, 0x5b, 0x77, 0xe3, 0x1e, 0x24,
	0x20, 0xdd, 0x25, 0xe9, 0x41, 0xec, 0x45, 0x8c, 0x58, 0x28, 0x18, 0x08, 0x03, 0xbd, 0x78, 0x09,
	0x93, 0xc9, 0x9b, 0xed, 0x90, 0x3f, 0xb3, 0xcc, 0x4c, 0xa2, 0xf9, 0x00, 0x82, 0x47, 0x0f, 0x82,
	0x1e, 0xfb, 0x71, 0x3c, 0xf6, 0xe8, 0x29, 0x48, 0x72, 0xf1, 0x9c, 0x4f, 0x20, 0xb3, 0x9b, 0x94,
	0xa5, 0x69, 0x82, 0xb9, 0xcd, 0xcb, 0xfe, 0x9e, 0x67, 0x9e, 0x9d, 0xf7, 0x7d, 0x51, 0x99, 0xb7,
	0x59, 0x40, 0xa3, 0xa8, 0xcf, 0x19, 0xd5, 0x5c, 0x0c, 0x55, 0xd0, 0x05, 0x08, 0xc6, 0xd5, 0x40,
	0x7f, 0xf6, 0x23, 0x29, 0xb4, 0xc0, 0x4f, 0x79, 0x9b, 0xf9, 0x69, 0xc2, 0xef, 0x02, 0xf8, 0xe3,
	0xaa, 0x5d, 0x0c, 0x45, 0x28, 0x62, 0x26, 0x30, 0xa7, 0x04, 0xb7, 0x9f, 0x6f, 0x32, 0x34, 0xaa,
	0x14, 0xc2, 0x84, 0x84, 0x80, 0x5d, 0xd2, 0xe1, 0x10, 0xfa, 0xe6, 0xf3, 0xf2, 0x98, 0x20, 0xde,
	0x4f, 0x0b, 0x39, 0x0d, 0x15, 0x12, 0x08, 0xb9, 0xd2, 0x20, 0xdf, 0x89, 0xd1, 0x50, 0x83, 0x8c,
	0xa8, 0xd4, 0x93, 0xb7, 0x9d, 0x8e, 0x04, 0xa5, 0x70, 0x09, 0x1d, 0xd0, 0xe4, 0x58, 0xb2, 0xca,
	0x56, 0x25, 0x4f, 0x56, 0x25, 0x26, 0xa8, 0xc8, 0x52, 0x82, 0xd6, 0x0a, 0xdb, 0x33, 0x58, 0xdd,
	0x5d, 0x4c, 0xdd, 0xc3, 0x09, 0x1d, 0xf4, 0x4f, 0xbd, 0xbb, 0x28, 0x8f, 0x3c, 0x66, 0xeb, 0xb7,
	0x9d, 0xe6, 0xbe, 0x5e, 0xb9, 0x99, 0xbf, 0x57, 0x6e, 0xc6, 0xab, 0xa0, 0x17, 0xeb, 0xc9, 0x9a,
	0x29, 0x96, 0x80, 0x8a, 0xc4, 0x50, 0x81, 0xf7, 0x25, 0x8b, 0x70, 0x43, 0x85, 0xef, 0x15, 0x93,
	0xe2, 0x53, 0x93, 0xb2, 0x1e, 0xe8, 0x33, 0x00, 0xdc, 0x44, 0xf9, 0x28, 0x2e, 0x5a, 0xbc, 0x13,
	0x47, 0x2f, 0xd4, 0x9e, 0xf9, 0xe6, 0x91, 0xcd, 0x93, 0xf8, 0xab, 0x77, 0x18, 0x57, 0xfd, 0x44,
	0x72, 0xde, 0xa9, 0x17, 0x17, 0x53, 0xf7, 0x51, 0x12, 0xf9, 0x46, 0xe9, 0x91, 0x5c, 0xb4, 0xfc,
	0x8e, 0x2f, 0x50, 0x41, 0x02, 0x03, 0x3e, 0x86, 0x56, 0x17, 0x20, 0xfe, 0xcf, 0x42, 0xed, 0xc8,
	0xdf, 0xd0, 0x38, 0xff, 0x0c, 0xa0, 0xfe, 0x64, 0x31, 0x75, 0x71, 0x62, 0x99, 0x92, 0x7a, 0x04,
	0x2d, 0x2b, 0x13, 0xf4, 0x1c, 0x1d, 0x50, 0xd6, 0x8b, 0x2d, 0xb3, 0xff, 0x61, 0x89, 0x17, 0x53,
	0xf7, 0x41, 0x62, 0xb9, 0x94, 0x79, 0x64, 0x9f, 0xb2, 0x9e, 0xb1, 0xba, 0x40, 0x05, 0xcd, 0x07,
	0x20, 0x46, 0x3a, 0xb6, 0xbb, 0xb7, 0x5b, 0xc2, 0x94, 0xd4, 0x23, 0x68, 0x59, 0x19, 0x5b, 0x1b,
	0xe5, 0x24, 0xf4, 0xe9, 0x04, 0xa4, 0x2a, 0xdd, 0x2f, 0x67, 0x2b, 0x79, 0x72, 0x53, 0xa7, 0x3a,
	0x76, 0x84, 0xec, 0xf5, 0x36, 0xac, 0xba, 0x54, 0xfb, 0xbe, 0x87, 0xb2, 0x0d, 0x15, 0xe2, 0x1f,
	0x16, 0x3a, 0xdc, 0xd2, 0x55, 0xfc, 0x6a, 0x63, 0xda, 0xed, 0x83, 0x6a, 0xbf, 0xd9, 0x41, 0x78,
	0xd7, 0x1c, 0x61, 0x85, 0x1e, 0xde, 0x9e, 0xa1, 0x97, 0xdb, 0x3c, 0x6f, 0xc1, 0xf6, 0xc9, 0x0e,
	0xf0, 0xea, 0xd2, 0xfa, 0x87, 0x5f, 0x33, 0xc7, 0xba, 0x9e, 0x39, 0xd6, 0x9f, 0x99, 0x63, 0x7d,
	0x9b, 0x3b, 0x99, 0xeb, 0xb9, 0x93, 0xf9, 0x3d, 0x77, 0x32, 0x1f, 0x6b, 0x21, 0xd7, 0x97, 0xa3,
	0xb6, 0xcf, 0xc4, 0x20, 0x60, 0x42, 0x0d, 0x84, 0x0a, 0x78, 0x9b, 0x1d, 0x87, 0x22, 0x18, 0x88,
	0xce, 0xa8, 0x0f, 0xca, 0xac, 0xbf, 0x0a, 0x6a, 0xaf, 0x8f, 0xcd, 0xe6, 0xeb, 0x49, 0x04, 0xaa,
	0xbd, 0x1f, 0xaf, 0xf5, 0xc9, 0xbf, 0x00, 0x00, 0x00, 0xff, 0xff, 0xe1, 0xc9, 0x06, 0xc0, 0x6f,
	0x04, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// MsgClient is the client API for Msg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type MsgClient interface {
	// RegisterCounterpartyAddress defines a rpc handler method for MsgRegisterCounterpartyAddress
	// RegisterCounterpartyAddress is called by the relayer on each channelEnd and allows them to specify their
	// counterparty address before relaying. This ensures they will be properly compensated for forward relaying since
	// destination chain must send back relayer's source address (counterparty address) in acknowledgement. This function
	// may be called more than once by a relayer, in which case, latest counterparty address is always used.
	RegisterCounterPartyAddress(ctx context.Context, in *MsgRegisterCounterpartyAddress, opts ...grpc.CallOption) (*MsgRegisterCounterPartyAddressResponse, error)
	// EscrowPacketFee defines a rpc handler method for MsgEscrowPacketFee
	// EscrowPacketFee is an open callback that may be called by any module/user that wishes to escrow funds in order to
	// incentivize the relaying of the given packet.
	EscrowPacketFee(ctx context.Context, in *MsgEscrowPacketFee, opts ...grpc.CallOption) (*MsgEscrowPacketFeeResponse, error)
}

type msgClient struct {
	cc grpc1.ClientConn
}

func NewMsgClient(cc grpc1.ClientConn) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) RegisterCounterPartyAddress(ctx context.Context, in *MsgRegisterCounterpartyAddress, opts ...grpc.CallOption) (*MsgRegisterCounterPartyAddressResponse, error) {
	out := new(MsgRegisterCounterPartyAddressResponse)
	err := c.cc.Invoke(ctx, "/ibc.applications.fee.v1.Msg/RegisterCounterPartyAddress", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) EscrowPacketFee(ctx context.Context, in *MsgEscrowPacketFee, opts ...grpc.CallOption) (*MsgEscrowPacketFeeResponse, error) {
	out := new(MsgEscrowPacketFeeResponse)
	err := c.cc.Invoke(ctx, "/ibc.applications.fee.v1.Msg/EscrowPacketFee", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
type MsgServer interface {
	// RegisterCounterpartyAddress defines a rpc handler method for MsgRegisterCounterpartyAddress
	// RegisterCounterpartyAddress is called by the relayer on each channelEnd and allows them to specify their
	// counterparty address before relaying. This ensures they will be properly compensated for forward relaying since
	// destination chain must send back relayer's source address (counterparty address) in acknowledgement. This function
	// may be called more than once by a relayer, in which case, latest counterparty address is always used.
	RegisterCounterPartyAddress(context.Context, *MsgRegisterCounterpartyAddress) (*MsgRegisterCounterPartyAddressResponse, error)
	// EscrowPacketFee defines a rpc handler method for MsgEscrowPacketFee
	// EscrowPacketFee is an open callback that may be called by any module/user that wishes to escrow funds in order to
	// incentivize the relaying of the given packet.
	EscrowPacketFee(context.Context, *MsgEscrowPacketFee) (*MsgEscrowPacketFeeResponse, error)
}

// UnimplementedMsgServer can be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (*UnimplementedMsgServer) RegisterCounterPartyAddress(ctx context.Context, req *MsgRegisterCounterpartyAddress) (*MsgRegisterCounterPartyAddressResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterCounterPartyAddress not implemented")
}
func (*UnimplementedMsgServer) EscrowPacketFee(ctx context.Context, req *MsgEscrowPacketFee) (*MsgEscrowPacketFeeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method EscrowPacketFee not implemented")
}

func RegisterMsgServer(s grpc1.Server, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

func _Msg_RegisterCounterPartyAddress_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRegisterCounterpartyAddress)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RegisterCounterPartyAddress(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ibc.applications.fee.v1.Msg/RegisterCounterPartyAddress",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RegisterCounterPartyAddress(ctx, req.(*MsgRegisterCounterpartyAddress))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_EscrowPacketFee_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgEscrowPacketFee)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).EscrowPacketFee(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ibc.applications.fee.v1.Msg/EscrowPacketFee",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).EscrowPacketFee(ctx, req.(*MsgEscrowPacketFee))
	}
	return interceptor(ctx, in, info, handler)
}

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "ibc.applications.fee.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RegisterCounterPartyAddress",
			Handler:    _Msg_RegisterCounterPartyAddress_Handler,
		},
		{
			MethodName: "EscrowPacketFee",
			Handler:    _Msg_EscrowPacketFee_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "ibc/applications/fee/v1/tx.proto",
}

func (m *MsgRegisterCounterpartyAddress) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgRegisterCounterpartyAddress) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgRegisterCounterpartyAddress) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.CounterpartyAddress) > 0 {
		i -= len(m.CounterpartyAddress)
		copy(dAtA[i:], m.CounterpartyAddress)
		i = encodeVarintTx(dAtA, i, uint64(len(m.CounterpartyAddress)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Address) > 0 {
		i -= len(m.Address)
		copy(dAtA[i:], m.Address)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Address)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgRegisterCounterPartyAddressResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgRegisterCounterPartyAddressResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgRegisterCounterPartyAddressResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *MsgEscrowPacketFee) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgEscrowPacketFee) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgEscrowPacketFee) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Relayers) > 0 {
		for iNdEx := len(m.Relayers) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Relayers[iNdEx])
			copy(dAtA[i:], m.Relayers[iNdEx])
			i = encodeVarintTx(dAtA, i, uint64(len(m.Relayers[iNdEx])))
			i--
			dAtA[i] = 0x2a
		}
	}
	if m.TimeoutFee != nil {
		{
			size, err := m.TimeoutFee.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintTx(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x22
	}
	if m.AckFee != nil {
		{
			size, err := m.AckFee.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintTx(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x1a
	}
	if m.ReceiveFee != nil {
		{
			size, err := m.ReceiveFee.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintTx(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	if m.PacketId != nil {
		{
			size, err := m.PacketId.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintTx(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgEscrowPacketFeeResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgEscrowPacketFeeResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgEscrowPacketFeeResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func encodeVarintTx(dAtA []byte, offset int, v uint64) int {
	offset -= sovTx(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MsgRegisterCounterpartyAddress) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Address)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.CounterpartyAddress)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}

func (m *MsgRegisterCounterPartyAddressResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *MsgEscrowPacketFee) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.PacketId != nil {
		l = m.PacketId.Size()
		n += 1 + l + sovTx(uint64(l))
	}
	if m.ReceiveFee != nil {
		l = m.ReceiveFee.Size()
		n += 1 + l + sovTx(uint64(l))
	}
	if m.AckFee != nil {
		l = m.AckFee.Size()
		n += 1 + l + sovTx(uint64(l))
	}
	if m.TimeoutFee != nil {
		l = m.TimeoutFee.Size()
		n += 1 + l + sovTx(uint64(l))
	}
	if len(m.Relayers) > 0 {
		for _, s := range m.Relayers {
			l = len(s)
			n += 1 + l + sovTx(uint64(l))
		}
	}
	return n
}

func (m *MsgEscrowPacketFeeResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func sovTx(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTx(x uint64) (n int) {
	return sovTx(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MsgRegisterCounterpartyAddress) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgRegisterCounterpartyAddress: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgRegisterCounterpartyAddress: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Address", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Address = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CounterpartyAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.CounterpartyAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *MsgRegisterCounterPartyAddressResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgRegisterCounterPartyAddressResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgRegisterCounterPartyAddressResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *MsgEscrowPacketFee) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgEscrowPacketFee: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgEscrowPacketFee: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field PacketId", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.PacketId == nil {
				m.PacketId = &types.PacketId{}
			}
			if err := m.PacketId.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ReceiveFee", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.ReceiveFee == nil {
				m.ReceiveFee = &Fee{}
			}
			if err := m.ReceiveFee.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AckFee", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.AckFee == nil {
				m.AckFee = &Fee{}
			}
			if err := m.AckFee.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TimeoutFee", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.TimeoutFee == nil {
				m.TimeoutFee = &Fee{}
			}
			if err := m.TimeoutFee.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Relayers", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Relayers = append(m.Relayers, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *MsgEscrowPacketFeeResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgEscrowPacketFeeResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgEscrowPacketFeeResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipTx(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTx
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowTx
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowTx
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthTx
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTx
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTx
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTx        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTx          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTx = fmt.Errorf("proto: unexpected end of group")
)
