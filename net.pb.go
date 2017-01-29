// Code generated by protoc-gen-go.
// source: net.proto
// DO NOT EDIT!

/*
Package chord is a generated protocol buffer package.

It is generated from these files:
	net.proto

It has these top-level messages:
	Payload
*/
package chord

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Payload struct {
	Data []byte `protobuf:"bytes,1,opt,name=Data,json=data,proto3" json:"Data,omitempty"`
}

func (m *Payload) Reset()                    { *m = Payload{} }
func (m *Payload) String() string            { return proto.CompactTextString(m) }
func (*Payload) ProtoMessage()               {}
func (*Payload) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Payload) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func init() {
	proto.RegisterType((*Payload)(nil), "chord.Payload")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for Chord service

type ChordClient interface {
	ListVnodesServe(ctx context.Context, in *Payload, opts ...grpc.CallOption) (*Payload, error)
	PingServe(ctx context.Context, in *Payload, opts ...grpc.CallOption) (*Payload, error)
	NotifyServe(ctx context.Context, in *Payload, opts ...grpc.CallOption) (*Payload, error)
	GetPredecessorServe(ctx context.Context, in *Payload, opts ...grpc.CallOption) (*Payload, error)
	FindSuccessorsServe(ctx context.Context, in *Payload, opts ...grpc.CallOption) (*Payload, error)
	ClearPredecessorServe(ctx context.Context, in *Payload, opts ...grpc.CallOption) (*Payload, error)
	SkipSuccessorServe(ctx context.Context, in *Payload, opts ...grpc.CallOption) (*Payload, error)
}

type chordClient struct {
	cc *grpc.ClientConn
}

func NewChordClient(cc *grpc.ClientConn) ChordClient {
	return &chordClient{cc}
}

func (c *chordClient) ListVnodesServe(ctx context.Context, in *Payload, opts ...grpc.CallOption) (*Payload, error) {
	out := new(Payload)
	err := grpc.Invoke(ctx, "/chord.chord/ListVnodesServe", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *chordClient) PingServe(ctx context.Context, in *Payload, opts ...grpc.CallOption) (*Payload, error) {
	out := new(Payload)
	err := grpc.Invoke(ctx, "/chord.chord/PingServe", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *chordClient) NotifyServe(ctx context.Context, in *Payload, opts ...grpc.CallOption) (*Payload, error) {
	out := new(Payload)
	err := grpc.Invoke(ctx, "/chord.chord/NotifyServe", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *chordClient) GetPredecessorServe(ctx context.Context, in *Payload, opts ...grpc.CallOption) (*Payload, error) {
	out := new(Payload)
	err := grpc.Invoke(ctx, "/chord.chord/GetPredecessorServe", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *chordClient) FindSuccessorsServe(ctx context.Context, in *Payload, opts ...grpc.CallOption) (*Payload, error) {
	out := new(Payload)
	err := grpc.Invoke(ctx, "/chord.chord/FindSuccessorsServe", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *chordClient) ClearPredecessorServe(ctx context.Context, in *Payload, opts ...grpc.CallOption) (*Payload, error) {
	out := new(Payload)
	err := grpc.Invoke(ctx, "/chord.chord/ClearPredecessorServe", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *chordClient) SkipSuccessorServe(ctx context.Context, in *Payload, opts ...grpc.CallOption) (*Payload, error) {
	out := new(Payload)
	err := grpc.Invoke(ctx, "/chord.chord/SkipSuccessorServe", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Chord service

type ChordServer interface {
	ListVnodesServe(context.Context, *Payload) (*Payload, error)
	PingServe(context.Context, *Payload) (*Payload, error)
	NotifyServe(context.Context, *Payload) (*Payload, error)
	GetPredecessorServe(context.Context, *Payload) (*Payload, error)
	FindSuccessorsServe(context.Context, *Payload) (*Payload, error)
	ClearPredecessorServe(context.Context, *Payload) (*Payload, error)
	SkipSuccessorServe(context.Context, *Payload) (*Payload, error)
}

func RegisterChordServer(s *grpc.Server, srv ChordServer) {
	s.RegisterService(&_Chord_serviceDesc, srv)
}

func _Chord_ListVnodesServe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Payload)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ChordServer).ListVnodesServe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/chord.chord/ListVnodesServe",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ChordServer).ListVnodesServe(ctx, req.(*Payload))
	}
	return interceptor(ctx, in, info, handler)
}

func _Chord_PingServe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Payload)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ChordServer).PingServe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/chord.chord/PingServe",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ChordServer).PingServe(ctx, req.(*Payload))
	}
	return interceptor(ctx, in, info, handler)
}

func _Chord_NotifyServe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Payload)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ChordServer).NotifyServe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/chord.chord/NotifyServe",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ChordServer).NotifyServe(ctx, req.(*Payload))
	}
	return interceptor(ctx, in, info, handler)
}

func _Chord_GetPredecessorServe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Payload)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ChordServer).GetPredecessorServe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/chord.chord/GetPredecessorServe",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ChordServer).GetPredecessorServe(ctx, req.(*Payload))
	}
	return interceptor(ctx, in, info, handler)
}

func _Chord_FindSuccessorsServe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Payload)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ChordServer).FindSuccessorsServe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/chord.chord/FindSuccessorsServe",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ChordServer).FindSuccessorsServe(ctx, req.(*Payload))
	}
	return interceptor(ctx, in, info, handler)
}

func _Chord_ClearPredecessorServe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Payload)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ChordServer).ClearPredecessorServe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/chord.chord/ClearPredecessorServe",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ChordServer).ClearPredecessorServe(ctx, req.(*Payload))
	}
	return interceptor(ctx, in, info, handler)
}

func _Chord_SkipSuccessorServe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Payload)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ChordServer).SkipSuccessorServe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/chord.chord/SkipSuccessorServe",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ChordServer).SkipSuccessorServe(ctx, req.(*Payload))
	}
	return interceptor(ctx, in, info, handler)
}

var _Chord_serviceDesc = grpc.ServiceDesc{
	ServiceName: "chord.chord",
	HandlerType: (*ChordServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ListVnodesServe",
			Handler:    _Chord_ListVnodesServe_Handler,
		},
		{
			MethodName: "PingServe",
			Handler:    _Chord_PingServe_Handler,
		},
		{
			MethodName: "NotifyServe",
			Handler:    _Chord_NotifyServe_Handler,
		},
		{
			MethodName: "GetPredecessorServe",
			Handler:    _Chord_GetPredecessorServe_Handler,
		},
		{
			MethodName: "FindSuccessorsServe",
			Handler:    _Chord_FindSuccessorsServe_Handler,
		},
		{
			MethodName: "ClearPredecessorServe",
			Handler:    _Chord_ClearPredecessorServe_Handler,
		},
		{
			MethodName: "SkipSuccessorServe",
			Handler:    _Chord_SkipSuccessorServe_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "net.proto",
}

func init() { proto.RegisterFile("net.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 191 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0xe2, 0xcc, 0x4b, 0x2d, 0xd1,
	0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x4d, 0xce, 0xc8, 0x2f, 0x4a, 0x51, 0x92, 0xe5, 0x62,
	0x0f, 0x48, 0xac, 0xcc, 0xc9, 0x4f, 0x4c, 0x11, 0x12, 0xe2, 0x62, 0x71, 0x49, 0x2c, 0x49, 0x94,
	0x60, 0x54, 0x60, 0xd4, 0xe0, 0x09, 0x62, 0x49, 0x49, 0x2c, 0x49, 0x34, 0x6a, 0x64, 0xe6, 0x82,
	0x28, 0x14, 0x32, 0xe6, 0xe2, 0xf7, 0xc9, 0x2c, 0x2e, 0x09, 0xcb, 0xcb, 0x4f, 0x49, 0x2d, 0x0e,
	0x4e, 0x2d, 0x2a, 0x4b, 0x15, 0xe2, 0xd3, 0x03, 0x4b, 0xe9, 0x41, 0x0d, 0x90, 0x42, 0xe3, 0x2b,
	0x31, 0x08, 0xe9, 0x72, 0x71, 0x06, 0x64, 0xe6, 0xa5, 0x13, 0xab, 0x5c, 0x9f, 0x8b, 0xdb, 0x2f,
	0xbf, 0x24, 0x33, 0xad, 0x92, 0x58, 0x0d, 0xe6, 0x5c, 0xc2, 0xee, 0xa9, 0x25, 0x01, 0x45, 0xa9,
	0x29, 0xa9, 0xc9, 0xa9, 0xc5, 0xc5, 0xf9, 0x45, 0x24, 0x68, 0x74, 0xcb, 0xcc, 0x4b, 0x09, 0x2e,
	0x4d, 0x86, 0xe8, 0x23, 0xda, 0x47, 0x96, 0x5c, 0xa2, 0xce, 0x39, 0xa9, 0x89, 0x45, 0x64, 0xd8,
	0x69, 0xc6, 0x25, 0x14, 0x9c, 0x9d, 0x59, 0x00, 0xb7, 0x93, 0x48, 0x7d, 0x49, 0x6c, 0xe0, 0x08,
	0x33, 0x06, 0x04, 0x00, 0x00, 0xff, 0xff, 0x07, 0x02, 0xe2, 0xbf, 0xbd, 0x01, 0x00, 0x00,
}
