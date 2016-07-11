// Code generated by protoc-gen-go.
// source: consensus.proto
// DO NOT EDIT!

/*
Package consensus is a generated protocol buffer package.

It is generated from these files:
	consensus.proto

It has these top-level messages:
	Message
	Block
*/
package consensus

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf "google/protobuf"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type Message struct {
	Data []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
}

func (m *Message) Reset()         { *m = Message{} }
func (m *Message) String() string { return proto.CompactTextString(m) }
func (*Message) ProtoMessage()    {}

type Block struct {
	Messages []*Message `protobuf:"bytes,1,rep,name=messages" json:"messages,omitempty"`
	PrevHash []byte     `protobuf:"bytes,2,opt,name=prev_hash,proto3" json:"prev_hash,omitempty"`
}

func (m *Block) Reset()         { *m = Block{} }
func (m *Block) String() string { return proto.CompactTextString(m) }
func (*Block) ProtoMessage()    {}

func (m *Block) GetMessages() []*Message {
	if m != nil {
		return m.Messages
	}
	return nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// Client API for AtomicBroadcast service

type AtomicBroadcastClient interface {
	Broadcast(ctx context.Context, in *Message, opts ...grpc.CallOption) (*google_protobuf.Empty, error)
	Deliver(ctx context.Context, in *google_protobuf.Empty, opts ...grpc.CallOption) (AtomicBroadcast_DeliverClient, error)
}

type atomicBroadcastClient struct {
	cc *grpc.ClientConn
}

func NewAtomicBroadcastClient(cc *grpc.ClientConn) AtomicBroadcastClient {
	return &atomicBroadcastClient{cc}
}

func (c *atomicBroadcastClient) Broadcast(ctx context.Context, in *Message, opts ...grpc.CallOption) (*google_protobuf.Empty, error) {
	out := new(google_protobuf.Empty)
	err := grpc.Invoke(ctx, "/consensus.atomic_broadcast/broadcast", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *atomicBroadcastClient) Deliver(ctx context.Context, in *google_protobuf.Empty, opts ...grpc.CallOption) (AtomicBroadcast_DeliverClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_AtomicBroadcast_serviceDesc.Streams[0], c.cc, "/consensus.atomic_broadcast/deliver", opts...)
	if err != nil {
		return nil, err
	}
	x := &atomicBroadcastDeliverClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type AtomicBroadcast_DeliverClient interface {
	Recv() (*Block, error)
	grpc.ClientStream
}

type atomicBroadcastDeliverClient struct {
	grpc.ClientStream
}

func (x *atomicBroadcastDeliverClient) Recv() (*Block, error) {
	m := new(Block)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Server API for AtomicBroadcast service

type AtomicBroadcastServer interface {
	Broadcast(context.Context, *Message) (*google_protobuf.Empty, error)
	Deliver(*google_protobuf.Empty, AtomicBroadcast_DeliverServer) error
}

func RegisterAtomicBroadcastServer(s *grpc.Server, srv AtomicBroadcastServer) {
	s.RegisterService(&_AtomicBroadcast_serviceDesc, srv)
}

func _AtomicBroadcast_Broadcast_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(Message)
	if err := dec(in); err != nil {
		return nil, err
	}
	out, err := srv.(AtomicBroadcastServer).Broadcast(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _AtomicBroadcast_Deliver_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(google_protobuf.Empty)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(AtomicBroadcastServer).Deliver(m, &atomicBroadcastDeliverServer{stream})
}

type AtomicBroadcast_DeliverServer interface {
	Send(*Block) error
	grpc.ServerStream
}

type atomicBroadcastDeliverServer struct {
	grpc.ServerStream
}

func (x *atomicBroadcastDeliverServer) Send(m *Block) error {
	return x.ServerStream.SendMsg(m)
}

var _AtomicBroadcast_serviceDesc = grpc.ServiceDesc{
	ServiceName: "consensus.atomic_broadcast",
	HandlerType: (*AtomicBroadcastServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "broadcast",
			Handler:    _AtomicBroadcast_Broadcast_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "deliver",
			Handler:       _AtomicBroadcast_Deliver_Handler,
			ServerStreams: true,
		},
	},
}