// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        v5.29.3
// source: clerk.proto

package pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Event struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            []byte                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Attributes    []*Attribute           `protobuf:"bytes,2,rep,name=attributes,proto3" json:"attributes,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Event) Reset() {
	*x = Event{}
	mi := &file_clerk_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Event) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Event) ProtoMessage() {}

func (x *Event) ProtoReflect() protoreflect.Message {
	mi := &file_clerk_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Event.ProtoReflect.Descriptor instead.
func (*Event) Descriptor() ([]byte, []int) {
	return file_clerk_proto_rawDescGZIP(), []int{0}
}

func (x *Event) GetId() []byte {
	if x != nil {
		return x.Id
	}
	return nil
}

func (x *Event) GetAttributes() []*Attribute {
	if x != nil {
		return x.Attributes
	}
	return nil
}

type Attribute struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	Key   string                 `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	// Types that are valid to be assigned to Value:
	//
	//	*Attribute_AsString
	//	*Attribute_AsInt64
	//	*Attribute_AsDouble
	//	*Attribute_AsBool
	Value         isAttribute_Value `protobuf_oneof:"value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Attribute) Reset() {
	*x = Attribute{}
	mi := &file_clerk_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Attribute) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Attribute) ProtoMessage() {}

func (x *Attribute) ProtoReflect() protoreflect.Message {
	mi := &file_clerk_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Attribute.ProtoReflect.Descriptor instead.
func (*Attribute) Descriptor() ([]byte, []int) {
	return file_clerk_proto_rawDescGZIP(), []int{1}
}

func (x *Attribute) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *Attribute) GetValue() isAttribute_Value {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *Attribute) GetAsString() string {
	if x != nil {
		if x, ok := x.Value.(*Attribute_AsString); ok {
			return x.AsString
		}
	}
	return ""
}

func (x *Attribute) GetAsInt64() int64 {
	if x != nil {
		if x, ok := x.Value.(*Attribute_AsInt64); ok {
			return x.AsInt64
		}
	}
	return 0
}

func (x *Attribute) GetAsDouble() float64 {
	if x != nil {
		if x, ok := x.Value.(*Attribute_AsDouble); ok {
			return x.AsDouble
		}
	}
	return 0
}

func (x *Attribute) GetAsBool() bool {
	if x != nil {
		if x, ok := x.Value.(*Attribute_AsBool); ok {
			return x.AsBool
		}
	}
	return false
}

type isAttribute_Value interface {
	isAttribute_Value()
}

type Attribute_AsString struct {
	AsString string `protobuf:"bytes,2,opt,name=as_string,json=asString,proto3,oneof"`
}

type Attribute_AsInt64 struct {
	AsInt64 int64 `protobuf:"varint,3,opt,name=as_int64,json=asInt64,proto3,oneof"`
}

type Attribute_AsDouble struct {
	AsDouble float64 `protobuf:"fixed64,4,opt,name=as_double,json=asDouble,proto3,oneof"`
}

type Attribute_AsBool struct {
	AsBool bool `protobuf:"varint,5,opt,name=as_bool,json=asBool,proto3,oneof"`
}

func (*Attribute_AsString) isAttribute_Value() {}

func (*Attribute_AsInt64) isAttribute_Value() {}

func (*Attribute_AsDouble) isAttribute_Value() {}

func (*Attribute_AsBool) isAttribute_Value() {}

type Index struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Chunks        []*Index_Chunk         `protobuf:"bytes,1,rep,name=chunks,proto3" json:"chunks,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Index) Reset() {
	*x = Index{}
	mi := &file_clerk_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Index) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Index) ProtoMessage() {}

func (x *Index) ProtoReflect() protoreflect.Message {
	mi := &file_clerk_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Index.ProtoReflect.Descriptor instead.
func (*Index) Descriptor() ([]byte, []int) {
	return file_clerk_proto_rawDescGZIP(), []int{2}
}

func (x *Index) GetChunks() []*Index_Chunk {
	if x != nil {
		return x.Chunks
	}
	return nil
}

type Bloom struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Chunks        []*Bloom_Chunk         `protobuf:"bytes,1,rep,name=chunks,proto3" json:"chunks,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Bloom) Reset() {
	*x = Bloom{}
	mi := &file_clerk_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Bloom) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Bloom) ProtoMessage() {}

func (x *Bloom) ProtoReflect() protoreflect.Message {
	mi := &file_clerk_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Bloom.ProtoReflect.Descriptor instead.
func (*Bloom) Descriptor() ([]byte, []int) {
	return file_clerk_proto_rawDescGZIP(), []int{3}
}

func (x *Bloom) GetChunks() []*Bloom_Chunk {
	if x != nil {
		return x.Chunks
	}
	return nil
}

type CreateEventsRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Events        []*Event               `protobuf:"bytes,1,rep,name=events,proto3" json:"events,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateEventsRequest) Reset() {
	*x = CreateEventsRequest{}
	mi := &file_clerk_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateEventsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateEventsRequest) ProtoMessage() {}

func (x *CreateEventsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_clerk_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateEventsRequest.ProtoReflect.Descriptor instead.
func (*CreateEventsRequest) Descriptor() ([]byte, []int) {
	return file_clerk_proto_rawDescGZIP(), []int{4}
}

func (x *CreateEventsRequest) GetEvents() []*Event {
	if x != nil {
		return x.Events
	}
	return nil
}

type CreateEventsResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateEventsResponse) Reset() {
	*x = CreateEventsResponse{}
	mi := &file_clerk_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateEventsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateEventsResponse) ProtoMessage() {}

func (x *CreateEventsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_clerk_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateEventsResponse.ProtoReflect.Descriptor instead.
func (*CreateEventsResponse) Descriptor() ([]byte, []int) {
	return file_clerk_proto_rawDescGZIP(), []int{5}
}

type SearchRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            []byte                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SearchRequest) Reset() {
	*x = SearchRequest{}
	mi := &file_clerk_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SearchRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SearchRequest) ProtoMessage() {}

func (x *SearchRequest) ProtoReflect() protoreflect.Message {
	mi := &file_clerk_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SearchRequest.ProtoReflect.Descriptor instead.
func (*SearchRequest) Descriptor() ([]byte, []int) {
	return file_clerk_proto_rawDescGZIP(), []int{6}
}

func (x *SearchRequest) GetId() []byte {
	if x != nil {
		return x.Id
	}
	return nil
}

type SearchResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Event         *Event                 `protobuf:"bytes,1,opt,name=event,proto3" json:"event,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SearchResponse) Reset() {
	*x = SearchResponse{}
	mi := &file_clerk_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SearchResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SearchResponse) ProtoMessage() {}

func (x *SearchResponse) ProtoReflect() protoreflect.Message {
	mi := &file_clerk_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SearchResponse.ProtoReflect.Descriptor instead.
func (*SearchResponse) Descriptor() ([]byte, []int) {
	return file_clerk_proto_rawDescGZIP(), []int{7}
}

func (x *SearchResponse) GetEvent() *Event {
	if x != nil {
		return x.Event
	}
	return nil
}

type AppendBlockRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Name          string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AppendBlockRequest) Reset() {
	*x = AppendBlockRequest{}
	mi := &file_clerk_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AppendBlockRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AppendBlockRequest) ProtoMessage() {}

func (x *AppendBlockRequest) ProtoReflect() protoreflect.Message {
	mi := &file_clerk_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AppendBlockRequest.ProtoReflect.Descriptor instead.
func (*AppendBlockRequest) Descriptor() ([]byte, []int) {
	return file_clerk_proto_rawDescGZIP(), []int{8}
}

func (x *AppendBlockRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type AppendBlockResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AppendBlockResponse) Reset() {
	*x = AppendBlockResponse{}
	mi := &file_clerk_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AppendBlockResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AppendBlockResponse) ProtoMessage() {}

func (x *AppendBlockResponse) ProtoReflect() protoreflect.Message {
	mi := &file_clerk_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AppendBlockResponse.ProtoReflect.Descriptor instead.
func (*AppendBlockResponse) Descriptor() ([]byte, []int) {
	return file_clerk_proto_rawDescGZIP(), []int{9}
}

type Index_Chunk struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Ids           [][]byte               `protobuf:"bytes,1,rep,name=ids,proto3" json:"ids,omitempty"`
	Size          int64                  `protobuf:"varint,2,opt,name=size,proto3" json:"size,omitempty"`
	Offset        int64                  `protobuf:"varint,3,opt,name=offset,proto3" json:"offset,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Index_Chunk) Reset() {
	*x = Index_Chunk{}
	mi := &file_clerk_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Index_Chunk) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Index_Chunk) ProtoMessage() {}

func (x *Index_Chunk) ProtoReflect() protoreflect.Message {
	mi := &file_clerk_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Index_Chunk.ProtoReflect.Descriptor instead.
func (*Index_Chunk) Descriptor() ([]byte, []int) {
	return file_clerk_proto_rawDescGZIP(), []int{2, 0}
}

func (x *Index_Chunk) GetIds() [][]byte {
	if x != nil {
		return x.Ids
	}
	return nil
}

func (x *Index_Chunk) GetSize() int64 {
	if x != nil {
		return x.Size
	}
	return 0
}

func (x *Index_Chunk) GetOffset() int64 {
	if x != nil {
		return x.Offset
	}
	return 0
}

type Bloom_Chunk struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Bloom         []byte                 `protobuf:"bytes,1,opt,name=bloom,proto3" json:"bloom,omitempty"`
	Size          int64                  `protobuf:"varint,2,opt,name=size,proto3" json:"size,omitempty"`
	Offset        int64                  `protobuf:"varint,3,opt,name=offset,proto3" json:"offset,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Bloom_Chunk) Reset() {
	*x = Bloom_Chunk{}
	mi := &file_clerk_proto_msgTypes[11]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Bloom_Chunk) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Bloom_Chunk) ProtoMessage() {}

func (x *Bloom_Chunk) ProtoReflect() protoreflect.Message {
	mi := &file_clerk_proto_msgTypes[11]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Bloom_Chunk.ProtoReflect.Descriptor instead.
func (*Bloom_Chunk) Descriptor() ([]byte, []int) {
	return file_clerk_proto_rawDescGZIP(), []int{3, 0}
}

func (x *Bloom_Chunk) GetBloom() []byte {
	if x != nil {
		return x.Bloom
	}
	return nil
}

func (x *Bloom_Chunk) GetSize() int64 {
	if x != nil {
		return x.Size
	}
	return 0
}

func (x *Bloom_Chunk) GetOffset() int64 {
	if x != nil {
		return x.Offset
	}
	return 0
}

var File_clerk_proto protoreflect.FileDescriptor

var file_clerk_proto_rawDesc = string([]byte{
	0x0a, 0x0b, 0x63, 0x6c, 0x65, 0x72, 0x6b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x09, 0x73,
	0x63, 0x72, 0x69, 0x62, 0x65, 0x2e, 0x76, 0x31, 0x22, 0x4d, 0x0a, 0x05, 0x45, 0x76, 0x65, 0x6e,
	0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x02, 0x69,
	0x64, 0x12, 0x34, 0x0a, 0x0a, 0x61, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x73, 0x18,
	0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x73, 0x63, 0x72, 0x69, 0x62, 0x65, 0x2e, 0x76,
	0x31, 0x2e, 0x41, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x52, 0x0a, 0x61, 0x74, 0x74,
	0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x73, 0x22, 0x9c, 0x01, 0x0a, 0x09, 0x41, 0x74, 0x74, 0x72,
	0x69, 0x62, 0x75, 0x74, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x1d, 0x0a, 0x09, 0x61, 0x73, 0x5f, 0x73, 0x74,
	0x72, 0x69, 0x6e, 0x67, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x08, 0x61, 0x73,
	0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x12, 0x1b, 0x0a, 0x08, 0x61, 0x73, 0x5f, 0x69, 0x6e, 0x74,
	0x36, 0x34, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x48, 0x00, 0x52, 0x07, 0x61, 0x73, 0x49, 0x6e,
	0x74, 0x36, 0x34, 0x12, 0x1d, 0x0a, 0x09, 0x61, 0x73, 0x5f, 0x64, 0x6f, 0x75, 0x62, 0x6c, 0x65,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x01, 0x48, 0x00, 0x52, 0x08, 0x61, 0x73, 0x44, 0x6f, 0x75, 0x62,
	0x6c, 0x65, 0x12, 0x19, 0x0a, 0x07, 0x61, 0x73, 0x5f, 0x62, 0x6f, 0x6f, 0x6c, 0x18, 0x05, 0x20,
	0x01, 0x28, 0x08, 0x48, 0x00, 0x52, 0x06, 0x61, 0x73, 0x42, 0x6f, 0x6f, 0x6c, 0x42, 0x07, 0x0a,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x7e, 0x0a, 0x05, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x12,
	0x2e, 0x0a, 0x06, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x16, 0x2e, 0x73, 0x63, 0x72, 0x69, 0x62, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x49, 0x6e, 0x64, 0x65,
	0x78, 0x2e, 0x43, 0x68, 0x75, 0x6e, 0x6b, 0x52, 0x06, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x73, 0x1a,
	0x45, 0x0a, 0x05, 0x43, 0x68, 0x75, 0x6e, 0x6b, 0x12, 0x10, 0x0a, 0x03, 0x69, 0x64, 0x73, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x03, 0x69, 0x64, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x69,
	0x7a, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x04, 0x73, 0x69, 0x7a, 0x65, 0x12, 0x16,
	0x0a, 0x06, 0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06,
	0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x22, 0x82, 0x01, 0x0a, 0x05, 0x42, 0x6c, 0x6f, 0x6f, 0x6d,
	0x12, 0x2e, 0x0a, 0x06, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x16, 0x2e, 0x73, 0x63, 0x72, 0x69, 0x62, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x6c, 0x6f,
	0x6f, 0x6d, 0x2e, 0x43, 0x68, 0x75, 0x6e, 0x6b, 0x52, 0x06, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x73,
	0x1a, 0x49, 0x0a, 0x05, 0x43, 0x68, 0x75, 0x6e, 0x6b, 0x12, 0x14, 0x0a, 0x05, 0x62, 0x6c, 0x6f,
	0x6f, 0x6d, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x62, 0x6c, 0x6f, 0x6f, 0x6d, 0x12,
	0x12, 0x0a, 0x04, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x04, 0x73,
	0x69, 0x7a, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x06, 0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x22, 0x3f, 0x0a, 0x13, 0x43,
	0x72, 0x65, 0x61, 0x74, 0x65, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x28, 0x0a, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x10, 0x2e, 0x73, 0x63, 0x72, 0x69, 0x62, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x45,
	0x76, 0x65, 0x6e, 0x74, 0x52, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x22, 0x16, 0x0a, 0x14,
	0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x22, 0x1f, 0x0a, 0x0d, 0x53, 0x65, 0x61, 0x72, 0x63, 0x68, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x02, 0x69, 0x64, 0x22, 0x38, 0x0a, 0x0e, 0x53, 0x65, 0x61, 0x72, 0x63, 0x68, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x26, 0x0a, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x73, 0x63, 0x72, 0x69, 0x62, 0x65, 0x2e,
	0x76, 0x31, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x52, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x22,
	0x28, 0x0a, 0x12, 0x41, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0x15, 0x0a, 0x13, 0x41, 0x70, 0x70,
	0x65, 0x6e, 0x64, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x32, 0x9e, 0x01, 0x0a, 0x08, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x65, 0x72, 0x12, 0x51, 0x0a,
	0x0c, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x12, 0x1e, 0x2e,
	0x73, 0x63, 0x72, 0x69, 0x62, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1f, 0x2e,
	0x73, 0x63, 0x72, 0x69, 0x62, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00,
	0x12, 0x3f, 0x0a, 0x06, 0x53, 0x65, 0x61, 0x72, 0x63, 0x68, 0x12, 0x18, 0x2e, 0x73, 0x63, 0x72,
	0x69, 0x62, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x65, 0x61, 0x72, 0x63, 0x68, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x19, 0x2e, 0x73, 0x63, 0x72, 0x69, 0x62, 0x65, 0x2e, 0x76, 0x31,
	0x2e, 0x53, 0x65, 0x61, 0x72, 0x63, 0x68, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22,
	0x00, 0x32, 0x9b, 0x01, 0x0a, 0x08, 0x53, 0x65, 0x61, 0x72, 0x63, 0x68, 0x65, 0x72, 0x12, 0x4e,
	0x0a, 0x0b, 0x41, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x12, 0x1d, 0x2e,
	0x73, 0x63, 0x72, 0x69, 0x62, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x41, 0x70, 0x70, 0x65, 0x6e, 0x64,
	0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1e, 0x2e, 0x73,
	0x63, 0x72, 0x69, 0x62, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x41, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x42,
	0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x3f,
	0x0a, 0x06, 0x53, 0x65, 0x61, 0x72, 0x63, 0x68, 0x12, 0x18, 0x2e, 0x73, 0x63, 0x72, 0x69, 0x62,
	0x65, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x65, 0x61, 0x72, 0x63, 0x68, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x19, 0x2e, 0x73, 0x63, 0x72, 0x69, 0x62, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x53,
	0x65, 0x61, 0x72, 0x63, 0x68, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42,
	0x07, 0x5a, 0x05, 0x2e, 0x2e, 0x2f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_clerk_proto_rawDescOnce sync.Once
	file_clerk_proto_rawDescData []byte
)

func file_clerk_proto_rawDescGZIP() []byte {
	file_clerk_proto_rawDescOnce.Do(func() {
		file_clerk_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_clerk_proto_rawDesc), len(file_clerk_proto_rawDesc)))
	})
	return file_clerk_proto_rawDescData
}

var file_clerk_proto_msgTypes = make([]protoimpl.MessageInfo, 12)
var file_clerk_proto_goTypes = []any{
	(*Event)(nil),                // 0: scribe.v1.Event
	(*Attribute)(nil),            // 1: scribe.v1.Attribute
	(*Index)(nil),                // 2: scribe.v1.Index
	(*Bloom)(nil),                // 3: scribe.v1.Bloom
	(*CreateEventsRequest)(nil),  // 4: scribe.v1.CreateEventsRequest
	(*CreateEventsResponse)(nil), // 5: scribe.v1.CreateEventsResponse
	(*SearchRequest)(nil),        // 6: scribe.v1.SearchRequest
	(*SearchResponse)(nil),       // 7: scribe.v1.SearchResponse
	(*AppendBlockRequest)(nil),   // 8: scribe.v1.AppendBlockRequest
	(*AppendBlockResponse)(nil),  // 9: scribe.v1.AppendBlockResponse
	(*Index_Chunk)(nil),          // 10: scribe.v1.Index.Chunk
	(*Bloom_Chunk)(nil),          // 11: scribe.v1.Bloom.Chunk
}
var file_clerk_proto_depIdxs = []int32{
	1,  // 0: scribe.v1.Event.attributes:type_name -> scribe.v1.Attribute
	10, // 1: scribe.v1.Index.chunks:type_name -> scribe.v1.Index.Chunk
	11, // 2: scribe.v1.Bloom.chunks:type_name -> scribe.v1.Bloom.Chunk
	0,  // 3: scribe.v1.CreateEventsRequest.events:type_name -> scribe.v1.Event
	0,  // 4: scribe.v1.SearchResponse.event:type_name -> scribe.v1.Event
	4,  // 5: scribe.v1.Recorder.CreateEvents:input_type -> scribe.v1.CreateEventsRequest
	6,  // 6: scribe.v1.Recorder.Search:input_type -> scribe.v1.SearchRequest
	8,  // 7: scribe.v1.Searcher.AppendBlock:input_type -> scribe.v1.AppendBlockRequest
	6,  // 8: scribe.v1.Searcher.Search:input_type -> scribe.v1.SearchRequest
	5,  // 9: scribe.v1.Recorder.CreateEvents:output_type -> scribe.v1.CreateEventsResponse
	7,  // 10: scribe.v1.Recorder.Search:output_type -> scribe.v1.SearchResponse
	9,  // 11: scribe.v1.Searcher.AppendBlock:output_type -> scribe.v1.AppendBlockResponse
	7,  // 12: scribe.v1.Searcher.Search:output_type -> scribe.v1.SearchResponse
	9,  // [9:13] is the sub-list for method output_type
	5,  // [5:9] is the sub-list for method input_type
	5,  // [5:5] is the sub-list for extension type_name
	5,  // [5:5] is the sub-list for extension extendee
	0,  // [0:5] is the sub-list for field type_name
}

func init() { file_clerk_proto_init() }
func file_clerk_proto_init() {
	if File_clerk_proto != nil {
		return
	}
	file_clerk_proto_msgTypes[1].OneofWrappers = []any{
		(*Attribute_AsString)(nil),
		(*Attribute_AsInt64)(nil),
		(*Attribute_AsDouble)(nil),
		(*Attribute_AsBool)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_clerk_proto_rawDesc), len(file_clerk_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   12,
			NumExtensions: 0,
			NumServices:   2,
		},
		GoTypes:           file_clerk_proto_goTypes,
		DependencyIndexes: file_clerk_proto_depIdxs,
		MessageInfos:      file_clerk_proto_msgTypes,
	}.Build()
	File_clerk_proto = out.File
	file_clerk_proto_goTypes = nil
	file_clerk_proto_depIdxs = nil
}
