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

	// google.protobuf.FieldMask and google.protobuf.Any are resolved against the
	// global registry (fieldmaskpb/anypb are imported by the tests), so the
	// dynamic fields share their canonical descriptors.
	fdp.Dependency = append(fdp.Dependency,
		"google/protobuf/field_mask.proto",
		"google/protobuf/any.proto",
	)
	fdp.MessageType = append(fdp.MessageType, extraMessages()...)

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

// --- field/message descriptor builders (shared by extraMessages) ----------

// scalarField builds an optional scalar field of the given proto type.
func scalarField(name string, num int32, t descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:   proto.String(name),
		Number: proto.Int32(num),
		Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:   t.Enum(),
	}
}

// msgField builds a message-typed field with the given label (optional/repeated).
func msgField(name string, num int32, typeName string, label descriptorpb.FieldDescriptorProto_Label) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(num),
		Label:    label.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
		TypeName: proto.String(typeName),
	}
}

// mapEntry builds a synthetic map-entry message with the given key/value types.
func mapEntry(name string, keyType, valType descriptorpb.FieldDescriptorProto_Type) *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{
		Name: proto.String(name),
		Field: []*descriptorpb.FieldDescriptorProto{
			scalarField("key", 1, keyType),
			scalarField("value", 2, valType),
		},
		Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)},
	}
}

// extraMessages returns additional message descriptors that exercise oneof,
// proto3 explicit presence, non-string map keys, well-known types (FieldMask,
// Any), numeric lexical edges, and the flow-leaf/literal-multiline interaction.
func extraMessages() []*descriptorpb.DescriptorProto {
	const (
		optional = descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
		repeated = descriptorpb.FieldDescriptorProto_LABEL_REPEATED
	)
	return []*descriptorpb.DescriptorProto{
		{
			// OneofMsg: a oneof with a scalar arm and a message arm.
			Name: proto.String("OneofMsg"),
			OneofDecl: []*descriptorpb.OneofDescriptorProto{
				{Name: proto.String("choice")},
			},
			Field: []*descriptorpb.FieldDescriptorProto{
				{
					Name:       proto.String("text_choice"),
					Number:     proto.Int32(1),
					Label:      optional.Enum(),
					Type:       descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					OneofIndex: proto.Int32(0),
				},
				{
					Name:       proto.String("msg_choice"),
					Number:     proto.Int32(2),
					Label:      optional.Enum(),
					Type:       descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName:   proto.String(".protoyaml_test.ChildLink"),
					OneofIndex: proto.Int32(0),
				},
			},
		},
		{
			// Presence: proto3 explicit presence (optional). Each optional field
			// is the sole member of a synthetic oneof named _<field>; those
			// synthetic oneofs must follow any real oneofs in OneofDecl.
			Name: proto.String("Presence"),
			OneofDecl: []*descriptorpb.OneofDescriptorProto{
				{Name: proto.String("_opt_i32")},
				{Name: proto.String("_opt_str")},
			},
			Field: []*descriptorpb.FieldDescriptorProto{
				{
					Name:           proto.String("opt_i32"),
					Number:         proto.Int32(1),
					Label:          optional.Enum(),
					Type:           descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
					OneofIndex:     proto.Int32(0),
					Proto3Optional: proto.Bool(true),
				},
				{
					Name:           proto.String("opt_str"),
					Number:         proto.Int32(2),
					Label:          optional.Enum(),
					Type:           descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					OneofIndex:     proto.Int32(1),
					Proto3Optional: proto.Bool(true),
				},
			},
		},
		{
			// Maps: non-string map keys (protojson encodes them as JSON strings).
			Name: proto.String("Maps"),
			Field: []*descriptorpb.FieldDescriptorProto{
				msgField("by_i64", 1, ".protoyaml_test.Maps.ByI64Entry", repeated),
				msgField("by_u32", 2, ".protoyaml_test.Maps.ByU32Entry", repeated),
				msgField("by_bool", 3, ".protoyaml_test.Maps.ByBoolEntry", repeated),
			},
			NestedType: []*descriptorpb.DescriptorProto{
				mapEntry("ByI64Entry", descriptorpb.FieldDescriptorProto_TYPE_INT64, descriptorpb.FieldDescriptorProto_TYPE_STRING),
				mapEntry("ByU32Entry", descriptorpb.FieldDescriptorProto_TYPE_UINT32, descriptorpb.FieldDescriptorProto_TYPE_STRING),
				mapEntry("ByBoolEntry", descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_TYPE_STRING),
			},
		},
		{
			// Wkt: google.protobuf.FieldMask and google.protobuf.Any fields.
			Name: proto.String("Wkt"),
			Field: []*descriptorpb.FieldDescriptorProto{
				msgField("mask", 1, ".google.protobuf.FieldMask", optional),
				msgField("payload", 2, ".google.protobuf.Any", optional),
			},
		},
		{
			// NumEdges: double + int32 for numeric lexical edge coverage.
			Name: proto.String("NumEdges"),
			Field: []*descriptorpb.FieldDescriptorProto{
				scalarField("d", 1, descriptorpb.FieldDescriptorProto_TYPE_DOUBLE),
				scalarField("i32", 2, descriptorpb.FieldDescriptorProto_TYPE_INT32),
			},
		},
		{
			// FlowLiteral: an all-scalar nested map (flow-leaf candidate) plus a
			// string field that can hold a multiline literal block.
			Name: proto.String("FlowLiteral"),
			Field: []*descriptorpb.FieldDescriptorProto{
				msgField("attrs", 1, ".protoyaml_test.FlowLiteral.AttrsEntry", repeated),
				scalarField("query", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING),
			},
			NestedType: []*descriptorpb.DescriptorProto{
				mapEntry("AttrsEntry", descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_TYPE_STRING),
			},
		},
	}
}
