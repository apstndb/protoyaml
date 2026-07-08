package protoyaml_test

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// buildTestFile constructs, entirely in code, a FileDescriptor describing
// plan-node-like messages that mirror the JSON field names used by Cloud
// Spanner's PlanNode. It deliberately avoids depending on cloud.google.com/go
// by building the descriptor dynamically. The google.protobuf.Struct fields are
// resolved against the global registry (structpb is imported by the tests), so
// their descriptors are identical to structpb's, letting real structpb.Struct
// values be assigned to dynamic fields.
func buildTestFile() protoreflect.FileDescriptor {
	int32Field := func(name string, num int32) *descriptorpb.FieldDescriptorProto {
		return &descriptorpb.FieldDescriptorProto{
			Name:   proto.String(name),
			Number: proto.Int32(num),
			Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
		}
	}
	stringField := func(name string, num int32) *descriptorpb.FieldDescriptorProto {
		return &descriptorpb.FieldDescriptorProto{
			Name:   proto.String(name),
			Number: proto.Int32(num),
			Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
		}
	}

	fdp := &descriptorpb.FileDescriptorProto{
		Name:       proto.String("protoyaml_test/plan.proto"),
		Package:    proto.String("protoyaml_test"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{"google/protobuf/struct.proto"},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: proto.String("Kind"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: proto.String("KIND_UNSPECIFIED"), Number: proto.Int32(0)},
					{Name: proto.String("RELATIONAL"), Number: proto.Int32(1)},
					{Name: proto.String("SCALAR"), Number: proto.Int32(2)},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("ChildLink"),
				Field: []*descriptorpb.FieldDescriptorProto{
					int32Field("child_index", 1),
					stringField("type", 2),
					stringField("variable", 3),
				},
			},
			{
				Name: proto.String("PlanNode"),
				Field: []*descriptorpb.FieldDescriptorProto{
					int32Field("index", 1),
					{
						Name:     proto.String("kind"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
						TypeName: proto.String(".protoyaml_test.Kind"),
					},
					stringField("display_name", 3),
					{
						Name:     proto.String("child_links"),
						Number:   proto.Int32(4),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".protoyaml_test.ChildLink"),
					},
					{
						Name:     proto.String("metadata"),
						Number:   proto.Int32(6),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".google.protobuf.Struct"),
					},
					{
						Name:     proto.String("execution_stats"),
						Number:   proto.Int32(7),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".google.protobuf.Struct"),
					},
				},
			},
			{
				// Scalars exercises numeric/bytes/enum/repeated/map JSON encodings.
				Name: proto.String("Scalars"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   proto.String("i64"),
						Number: proto.Int32(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
					},
					{
						Name:   proto.String("u64"),
						Number: proto.Int32(2),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum(),
					},
					{
						Name:   proto.String("data"),
						Number: proto.Int32(3),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum(),
					},
					{
						Name:   proto.String("d"),
						Number: proto.Int32(4),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum(),
					},
					{
						Name:     proto.String("kind"),
						Number:   proto.Int32(5),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
						TypeName: proto.String(".protoyaml_test.Kind"),
					},
					{
						Name:   proto.String("nums"),
						Number: proto.Int32(6),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
					},
					{
						Name:     proto.String("counts"),
						Number:   proto.Int32(7),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".protoyaml_test.Scalars.CountsEntry"),
					},
					{
						Name:   proto.String("text"),
						Number: proto.Int32(8),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
					{
						Name:   proto.String("b"),
						Number: proto.Int32(9),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum(),
					},
				},
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String("CountsEntry"),
						Field: []*descriptorpb.FieldDescriptorProto{
							stringField("key", 1),
							int32Field("value", 2),
						},
						Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)},
					},
				},
			},
		},
	}

	fd, err := protodesc.NewFile(fdp, protoregistry.GlobalFiles)
	if err != nil {
		panic(err)
	}
	return fd
}

// messageDescriptor returns the named top-level message descriptor from fd.
func messageDescriptor(fd protoreflect.FileDescriptor, name string) protoreflect.MessageDescriptor {
	md := fd.Messages().ByName(protoreflect.Name(name))
	if md == nil {
		panic("message not found: " + name)
	}
	return md
}
