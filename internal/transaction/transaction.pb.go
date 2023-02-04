// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.17.3
// source: internal/transaction/transaction.proto

package transaction

import (
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

// DataType is the transaction data type.
type DataType int32

const (
	DataType_UNKNOWN                    DataType = 0
	DataType_UPDATE_BLOCKCHAIN_SETTINGS DataType = 1
	DataType_CREATE_NODE                DataType = 2
	DataType_UPDATE_NODE                DataType = 3
	DataType_DATA_CONTRACT              DataType = 4
)

// Enum value maps for DataType.
var (
	DataType_name = map[int32]string{
		0: "UNKNOWN",
		1: "UPDATE_BLOCKCHAIN_SETTINGS",
		2: "CREATE_NODE",
		3: "UPDATE_NODE",
		4: "DATA_CONTRACT",
	}
	DataType_value = map[string]int32{
		"UNKNOWN":                    0,
		"UPDATE_BLOCKCHAIN_SETTINGS": 1,
		"CREATE_NODE":                2,
		"UPDATE_NODE":                3,
		"DATA_CONTRACT":              4,
	}
)

func (x DataType) Enum() *DataType {
	p := new(DataType)
	*p = x
	return p
}

func (x DataType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (DataType) Descriptor() protoreflect.EnumDescriptor {
	return file_internal_transaction_transaction_proto_enumTypes[0].Descriptor()
}

func (DataType) Type() protoreflect.EnumType {
	return &file_internal_transaction_transaction_proto_enumTypes[0]
}

func (x DataType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use DataType.Descriptor instead.
func (DataType) EnumDescriptor() ([]byte, []int) {
	return file_internal_transaction_transaction_proto_rawDescGZIP(), []int{0}
}

// ProtoTransaction is the proto representation of a transaction.
type ProtoTransaction struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// hash represents the block hash.
	Hash []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	// signature of the block.
	Signature []byte `protobuf:"bytes,2,opt,name=signature,proto3" json:"signature,omitempty"`
	// public_key is the sender's pubkey.
	PublicKey []byte `protobuf:"bytes,3,opt,name=public_key,json=publicKey,proto3" json:"public_key,omitempty"`
	// nounce of the transaction.
	Nounce []byte `protobuf:"bytes,4,opt,name=nounce,proto3" json:"nounce,omitempty"`
	// data attached to the transaction.
	Data []byte `protobuf:"bytes,5,opt,name=data,proto3" json:"data,omitempty"`
	// from sender
	From string `protobuf:"bytes,6,opt,name=from,proto3" json:"from,omitempty"`
	// to receiver.
	To string `protobuf:"bytes,7,opt,name=to,proto3" json:"to,omitempty"`
	// value of the transaction.
	Value string `protobuf:"bytes,8,opt,name=value,proto3" json:"value,omitempty"`
	// transaction_fees is the fees attached to a transaction.
	TransactionFees string `protobuf:"bytes,9,opt,name=transaction_fees,json=transactionFees,proto3" json:"transaction_fees,omitempty"`
	// chain represents the network chain.
	Chain []byte `protobuf:"bytes,10,opt,name=chain,proto3" json:"chain,omitempty"`
}

func (x *ProtoTransaction) Reset() {
	*x = ProtoTransaction{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_transaction_transaction_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProtoTransaction) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProtoTransaction) ProtoMessage() {}

func (x *ProtoTransaction) ProtoReflect() protoreflect.Message {
	mi := &file_internal_transaction_transaction_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProtoTransaction.ProtoReflect.Descriptor instead.
func (*ProtoTransaction) Descriptor() ([]byte, []int) {
	return file_internal_transaction_transaction_proto_rawDescGZIP(), []int{0}
}

func (x *ProtoTransaction) GetHash() []byte {
	if x != nil {
		return x.Hash
	}
	return nil
}

func (x *ProtoTransaction) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

func (x *ProtoTransaction) GetPublicKey() []byte {
	if x != nil {
		return x.PublicKey
	}
	return nil
}

func (x *ProtoTransaction) GetNounce() []byte {
	if x != nil {
		return x.Nounce
	}
	return nil
}

func (x *ProtoTransaction) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *ProtoTransaction) GetFrom() string {
	if x != nil {
		return x.From
	}
	return ""
}

func (x *ProtoTransaction) GetTo() string {
	if x != nil {
		return x.To
	}
	return ""
}

func (x *ProtoTransaction) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

func (x *ProtoTransaction) GetTransactionFees() string {
	if x != nil {
		return x.TransactionFees
	}
	return ""
}

func (x *ProtoTransaction) GetChain() []byte {
	if x != nil {
		return x.Chain
	}
	return nil
}

// DataPayload is the transaction data payload.
type DataPayload struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// type defines the payload type inside a transaction.
	Type DataType `protobuf:"varint,1,opt,name=type,proto3,enum=transaction.DataType" json:"type,omitempty"`
	// payload contains the byte array of the transaction data.
	Payload []byte `protobuf:"bytes,2,opt,name=payload,proto3" json:"payload,omitempty"`
}

func (x *DataPayload) Reset() {
	*x = DataPayload{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_transaction_transaction_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DataPayload) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DataPayload) ProtoMessage() {}

func (x *DataPayload) ProtoReflect() protoreflect.Message {
	mi := &file_internal_transaction_transaction_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DataPayload.ProtoReflect.Descriptor instead.
func (*DataPayload) Descriptor() ([]byte, []int) {
	return file_internal_transaction_transaction_proto_rawDescGZIP(), []int{1}
}

func (x *DataPayload) GetType() DataType {
	if x != nil {
		return x.Type
	}
	return DataType_UNKNOWN
}

func (x *DataPayload) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

var File_internal_transaction_transaction_proto protoreflect.FileDescriptor

var file_internal_transaction_transaction_proto_rawDesc = []byte{
	0x0a, 0x26, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x72, 0x61, 0x6e, 0x73,
	0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0b, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x8a, 0x02, 0x0a, 0x10, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x54,
	0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x61,
	0x73, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x68, 0x61, 0x73, 0x68, 0x12, 0x1c,
	0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x12, 0x1d, 0x0a, 0x0a,
	0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x5f, 0x6b, 0x65, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x09, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x12, 0x16, 0x0a, 0x06, 0x6e,
	0x6f, 0x75, 0x6e, 0x63, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x06, 0x6e, 0x6f, 0x75,
	0x6e, 0x63, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x12, 0x12, 0x0a, 0x04, 0x66, 0x72, 0x6f, 0x6d, 0x18,
	0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x66, 0x72, 0x6f, 0x6d, 0x12, 0x0e, 0x0a, 0x02, 0x74,
	0x6f, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x74, 0x6f, 0x12, 0x14, 0x0a, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x18, 0x08, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x12, 0x29, 0x0a, 0x10, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x5f, 0x66, 0x65, 0x65, 0x73, 0x18, 0x09, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0f, 0x74, 0x72, 0x61,
	0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x46, 0x65, 0x65, 0x73, 0x12, 0x14, 0x0a, 0x05,
	0x63, 0x68, 0x61, 0x69, 0x6e, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x63, 0x68, 0x61,
	0x69, 0x6e, 0x22, 0x52, 0x0a, 0x0b, 0x44, 0x61, 0x74, 0x61, 0x50, 0x61, 0x79, 0x6c, 0x6f, 0x61,
	0x64, 0x12, 0x29, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32,
	0x15, 0x2e, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x44, 0x61,
	0x74, 0x61, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x18, 0x0a, 0x07,
	0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x70,
	0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x2a, 0x6c, 0x0a, 0x08, 0x44, 0x61, 0x74, 0x61, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x0b, 0x0a, 0x07, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12,
	0x1e, 0x0a, 0x1a, 0x55, 0x50, 0x44, 0x41, 0x54, 0x45, 0x5f, 0x42, 0x4c, 0x4f, 0x43, 0x4b, 0x43,
	0x48, 0x41, 0x49, 0x4e, 0x5f, 0x53, 0x45, 0x54, 0x54, 0x49, 0x4e, 0x47, 0x53, 0x10, 0x01, 0x12,
	0x0f, 0x0a, 0x0b, 0x43, 0x52, 0x45, 0x41, 0x54, 0x45, 0x5f, 0x4e, 0x4f, 0x44, 0x45, 0x10, 0x02,
	0x12, 0x0f, 0x0a, 0x0b, 0x55, 0x50, 0x44, 0x41, 0x54, 0x45, 0x5f, 0x4e, 0x4f, 0x44, 0x45, 0x10,
	0x03, 0x12, 0x11, 0x0a, 0x0d, 0x44, 0x41, 0x54, 0x41, 0x5f, 0x43, 0x4f, 0x4e, 0x54, 0x52, 0x41,
	0x43, 0x54, 0x10, 0x04, 0x42, 0x37, 0x5a, 0x35, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x66, 0x69, 0x6c, 0x65, 0x66, 0x69, 0x6c, 0x65, 0x67, 0x6f, 0x2f, 0x66, 0x69,
	0x6c, 0x65, 0x66, 0x69, 0x6c, 0x65, 0x67, 0x6f, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61,
	0x6c, 0x2f, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_internal_transaction_transaction_proto_rawDescOnce sync.Once
	file_internal_transaction_transaction_proto_rawDescData = file_internal_transaction_transaction_proto_rawDesc
)

func file_internal_transaction_transaction_proto_rawDescGZIP() []byte {
	file_internal_transaction_transaction_proto_rawDescOnce.Do(func() {
		file_internal_transaction_transaction_proto_rawDescData = protoimpl.X.CompressGZIP(file_internal_transaction_transaction_proto_rawDescData)
	})
	return file_internal_transaction_transaction_proto_rawDescData
}

var file_internal_transaction_transaction_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_internal_transaction_transaction_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_internal_transaction_transaction_proto_goTypes = []interface{}{
	(DataType)(0),            // 0: transaction.DataType
	(*ProtoTransaction)(nil), // 1: transaction.ProtoTransaction
	(*DataPayload)(nil),      // 2: transaction.DataPayload
}
var file_internal_transaction_transaction_proto_depIdxs = []int32{
	0, // 0: transaction.DataPayload.type:type_name -> transaction.DataType
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_internal_transaction_transaction_proto_init() }
func file_internal_transaction_transaction_proto_init() {
	if File_internal_transaction_transaction_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_internal_transaction_transaction_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ProtoTransaction); i {
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
		file_internal_transaction_transaction_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DataPayload); i {
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
			RawDescriptor: file_internal_transaction_transaction_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_internal_transaction_transaction_proto_goTypes,
		DependencyIndexes: file_internal_transaction_transaction_proto_depIdxs,
		EnumInfos:         file_internal_transaction_transaction_proto_enumTypes,
		MessageInfos:      file_internal_transaction_transaction_proto_msgTypes,
	}.Build()
	File_internal_transaction_transaction_proto = out.File
	file_internal_transaction_transaction_proto_rawDesc = nil
	file_internal_transaction_transaction_proto_goTypes = nil
	file_internal_transaction_transaction_proto_depIdxs = nil
}
