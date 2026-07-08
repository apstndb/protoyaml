package protoyaml_test

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/apstndb/protoyaml"
	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// --- construction helpers -------------------------------------------------

func setField(m protoreflect.Message, name string, v protoreflect.Value) {
	fd := m.Descriptor().Fields().ByName(protoreflect.Name(name))
	if fd == nil {
		panic("field not found: " + name)
	}
	m.Set(fd, v)
}

func mustStruct(t *testing.T, fields map[string]any) *structpb.Struct {
	t.Helper()
	s, err := structpb.NewStruct(fields)
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}
	return s
}

// goldenPlanNode builds the PlanNode-like message used by the golden test.
func goldenPlanNode(t *testing.T, fd protoreflect.FileDescriptor) proto.Message {
	t.Helper()
	planMD := messageDescriptor(fd, "PlanNode")
	childMD := messageDescriptor(fd, "ChildLink")

	plan := dynamicpb.NewMessage(planMD)
	setField(plan.ProtoReflect(), "index", protoreflect.ValueOfInt32(1))
	setField(plan.ProtoReflect(), "kind", protoreflect.ValueOfEnum(protoreflect.EnumNumber(1))) // RELATIONAL
	setField(plan.ProtoReflect(), "display_name", protoreflect.ValueOfString("Unit Relation"))

	child := dynamicpb.NewMessage(childMD)
	setField(child.ProtoReflect(), "child_index", protoreflect.ValueOfInt32(2))
	list := plan.ProtoReflect().Mutable(planMD.Fields().ByName("child_links")).List()
	list.Append(protoreflect.ValueOfMessage(child.ProtoReflect()))

	metadata := mustStruct(t, map[string]any{"execution_method": "Row"})
	setField(plan.ProtoReflect(), "metadata", protoreflect.ValueOfMessage(metadata.ProtoReflect()))

	stats := mustStruct(t, map[string]any{
		"rows":              map[string]any{"total": "1", "unit": "rows"},
		"latency":           map[string]any{"total": "0", "unit": "msecs"},
		"cpu_time":          map[string]any{"total": "0", "unit": "msecs"},
		"execution_summary": map[string]any{"num_executions": "1"},
	})
	setField(plan.ProtoReflect(), "execution_stats", protoreflect.ValueOfMessage(stats.ProtoReflect()))

	return plan
}

// --- golden acceptance test ----------------------------------------------

func TestGoldenFlowLeafCollections(t *testing.T) {
	fd := buildTestFile()
	plan := goldenPlanNode(t, fd)

	got, err := protoyaml.Marshal(plan, protoyaml.WithFlowLeafCollections())
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	const want = `index: 1
kind: RELATIONAL
displayName: Unit Relation
childLinks:
- {childIndex: 2}
metadata: {execution_method: Row}
executionStats:
  cpu_time: {total: "0", unit: msecs}
  execution_summary: {num_executions: "1"}
  latency: {total: "0", unit: msecs}
  rows: {total: "1", unit: rows}
`
	if string(got) != want {
		t.Errorf("golden mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// TestFlowLeafRootStaysBlock: a message whose fields are all scalars would
// qualify as a leaf mapping, but the document root must stay block style so
// the output reads as a YAML document rather than an inline value.
func TestFlowLeafRootStaysBlock(t *testing.T) {
	fd := buildTestFile()
	planMD := messageDescriptor(fd, "PlanNode")

	plan := dynamicpb.NewMessage(planMD)
	setField(plan.ProtoReflect(), "index", protoreflect.ValueOfInt32(1))
	setField(plan.ProtoReflect(), "kind", protoreflect.ValueOfEnum(protoreflect.EnumNumber(1))) // RELATIONAL
	setField(plan.ProtoReflect(), "display_name", protoreflect.ValueOfString("Child"))

	got, err := protoyaml.Marshal(plan, protoyaml.WithFlowLeafCollections())
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	const want = `index: 1
kind: RELATIONAL
displayName: Child
`
	if string(got) != want {
		t.Errorf("root mapping must stay block:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// --- test message corpus --------------------------------------------------

// roundTripMessages returns messages safe for proto.Equal round-tripping
// (i.e. no NaN, which is never equal to itself).
func roundTripMessages(t *testing.T, fd protoreflect.FileDescriptor) map[string]proto.Message {
	t.Helper()
	scalarsMD := messageDescriptor(fd, "Scalars")

	scalars := dynamicpb.NewMessage(scalarsMD)
	setField(scalars.ProtoReflect(), "i64", protoreflect.ValueOfInt64(math.MaxInt64))
	setField(scalars.ProtoReflect(), "u64", protoreflect.ValueOfUint64(math.MaxUint64))
	setField(scalars.ProtoReflect(), "data", protoreflect.ValueOfBytes([]byte{0x00, 0x01, 0xfe, 0xff}))
	setField(scalars.ProtoReflect(), "d", protoreflect.ValueOfFloat64(3.5))
	setField(scalars.ProtoReflect(), "kind", protoreflect.ValueOfEnum(protoreflect.EnumNumber(2)))
	setField(scalars.ProtoReflect(), "b", protoreflect.ValueOfBool(true))
	nums := scalars.ProtoReflect().Mutable(scalarsMD.Fields().ByName("nums")).List()
	nums.Append(protoreflect.ValueOfInt64(1))
	nums.Append(protoreflect.ValueOfInt64(-2))
	nums.Append(protoreflect.ValueOfInt64(math.MinInt64))
	counts := scalars.ProtoReflect().Mutable(scalarsMD.Fields().ByName("counts")).Map()
	counts.Set(protoreflect.ValueOfString("a").MapKey(), protoreflect.ValueOfInt32(1))
	counts.Set(protoreflect.ValueOfString("b").MapKey(), protoreflect.ValueOfInt32(2))

	inf := dynamicpb.NewMessage(scalarsMD)
	setField(inf.ProtoReflect(), "d", protoreflect.ValueOfFloat64(math.Inf(1)))

	structVal, err := structpb.NewValue(map[string]any{
		"nested": []any{1.0, "two", true, nil},
		"n":      42.0,
	})
	if err != nil {
		t.Fatal(err)
	}
	listVal, err := structpb.NewList([]any{"x", 1.0, false})
	if err != nil {
		t.Fatal(err)
	}

	msgs := map[string]proto.Message{
		"planNode":     goldenPlanNode(t, fd),
		"scalars":      scalars,
		"scalarsInf":   inf,
		"emptyScalars": dynamicpb.NewMessage(scalarsMD),
		"struct":       mustStruct(t, map[string]any{"a": 1.0, "b": "s", "c": true, "d": nil}),
		"structValue":  structVal,
		"listValue":    listVal,
		"timestamp":    timestampAt(2021, 1, 2, 3, 4, 5),
		"duration":     durationpb.New(1500 * time.Millisecond), // 1.5s
		"wrapperInt64": wrapperspb.Int64(math.MaxInt64),
		"wrapperStr":   wrapperspb.String("hello"),
		"wrapperBool":  wrapperspb.Bool(true),
		"empty":        &emptypb.Empty{},
	}
	for name, m := range extraRoundTripMessages(t, fd) {
		msgs[name] = m
	}
	return msgs
}

// extraRoundTripMessages builds the corpus entries for the extended coverage:
// oneof (scalar + message arm), proto3 explicit presence (set-to-zero vs unset),
// non-string map keys, FieldMask, Any (WKT via default resolver), and numeric
// lexical edges that the goccy bridge handles faithfully. The three
// goccy-limited double forms (-0, 1e+21, 5e-324) are intentionally excluded from
// this shared corpus and covered separately by TestDoubleLexicalEdgeLimitations;
// see that test for the root-cause explanation. All entries here are finite and
// self-equal, so both property tests apply.
func extraRoundTripMessages(t *testing.T, fd protoreflect.FileDescriptor) map[string]proto.Message {
	t.Helper()
	childMD := messageDescriptor(fd, "ChildLink")
	oneofMD := messageDescriptor(fd, "OneofMsg")
	presMD := messageDescriptor(fd, "Presence")
	mapsMD := messageDescriptor(fd, "Maps")
	wktMD := messageDescriptor(fd, "Wkt")
	edgesMD := messageDescriptor(fd, "NumEdges")

	// oneof: scalar arm and message arm.
	oneofScalar := dynamicpb.NewMessage(oneofMD)
	setField(oneofScalar.ProtoReflect(), "text_choice", protoreflect.ValueOfString("picked"))
	oneofMsg := dynamicpb.NewMessage(oneofMD)
	oneofChild := dynamicpb.NewMessage(childMD)
	setField(oneofChild.ProtoReflect(), "child_index", protoreflect.ValueOfInt32(7))
	setField(oneofMsg.ProtoReflect(), "msg_choice", protoreflect.ValueOfMessage(oneofChild.ProtoReflect()))

	// proto3 optional: explicit zero vs unset must round-trip distinguishably.
	presZero := dynamicpb.NewMessage(presMD)
	setField(presZero.ProtoReflect(), "opt_i32", protoreflect.ValueOfInt32(0))
	setField(presZero.ProtoReflect(), "opt_str", protoreflect.ValueOfString(""))
	presUnset := dynamicpb.NewMessage(presMD)

	// non-string map keys.
	maps := dynamicpb.NewMessage(mapsMD)
	byI64 := maps.ProtoReflect().Mutable(mapsMD.Fields().ByName("by_i64")).Map()
	byI64.Set(protoreflect.ValueOfInt64(-5).MapKey(), protoreflect.ValueOfString("neg"))
	byI64.Set(protoreflect.ValueOfInt64(math.MaxInt64).MapKey(), protoreflect.ValueOfString("max"))
	byU32 := maps.ProtoReflect().Mutable(mapsMD.Fields().ByName("by_u32")).Map()
	byU32.Set(protoreflect.ValueOfUint32(math.MaxUint32).MapKey(), protoreflect.ValueOfString("umax"))
	byU32.Set(protoreflect.ValueOfUint32(0).MapKey(), protoreflect.ValueOfString("zero"))
	byBool := maps.ProtoReflect().Mutable(mapsMD.Fields().ByName("by_bool")).Map()
	byBool.Set(protoreflect.ValueOfBool(true).MapKey(), protoreflect.ValueOfString("t"))
	byBool.Set(protoreflect.ValueOfBool(false).MapKey(), protoreflect.ValueOfString("f"))

	// well-known types: FieldMask (camelCase, comma-joined) and Any (Duration).
	wkt := dynamicpb.NewMessage(wktMD)
	mask := &fieldmaskpb.FieldMask{Paths: []string{"foo_bar", "baz_qux"}}
	setField(wkt.ProtoReflect(), "mask", protoreflect.ValueOfMessage(mask.ProtoReflect()))
	anyDur, err := anypb.New(durationpb.New(1500 * time.Millisecond))
	if err != nil {
		t.Fatalf("anypb.New: %v", err)
	}
	setField(wkt.ProtoReflect(), "payload", protoreflect.ValueOfMessage(anyDur.ProtoReflect()))

	edge := func(set func(m protoreflect.Message)) proto.Message {
		m := dynamicpb.NewMessage(edgesMD)
		set(m.ProtoReflect())
		return m
	}
	setD := func(v float64) func(protoreflect.Message) {
		return func(m protoreflect.Message) { setField(m, "d", protoreflect.ValueOfFloat64(v)) }
	}
	setI32 := func(v int32) func(protoreflect.Message) {
		return func(m protoreflect.Message) { setField(m, "i32", protoreflect.ValueOfInt32(v)) }
	}

	return map[string]proto.Message{
		"oneofScalar":   oneofScalar,
		"oneofMsg":      oneofMsg,
		"presenceZero":  presZero,
		"presenceUnset": presUnset,
		"maps":          maps,
		"wkt":           wkt,
		"edgeMaxF64":    edge(setD(math.MaxFloat64)),
		"edgeI32Min":    edge(setI32(math.MinInt32)),
		"edgeI32Max":    edge(setI32(math.MaxInt32)),
	}
}

func timestampAt(year int, month time.Month, day, hour, min, sec int) *timestamppb.Timestamp {
	return timestamppb.New(time.Date(year, month, day, hour, min, sec, 0, time.UTC))
}

// --- conformance property (semantic equality) -----------------------------

func TestConformanceSemanticEquality(t *testing.T) {
	fd := buildTestFile()
	msgs := roundTripMessages(t, fd)

	// Add NaN cases here: they break proto.Equal but conform semantically.
	scalarsMD := messageDescriptor(fd, "Scalars")
	nan := dynamicpb.NewMessage(scalarsMD)
	setField(nan.ProtoReflect(), "d", protoreflect.ValueOfFloat64(math.NaN()))
	msgs["scalarsNaN"] = nan
	ninf := dynamicpb.NewMessage(scalarsMD)
	setField(ninf.ProtoReflect(), "d", protoreflect.ValueOfFloat64(math.Inf(-1)))
	msgs["scalarsNegInf"] = ninf

	for name, m := range msgs {
		for _, flow := range []bool{false, true} {
			t.Run(name+flowSuffix(flow), func(t *testing.T) {
				var opts []protoyaml.Option
				if flow {
					opts = append(opts, protoyaml.WithFlowLeafCollections())
				}
				y, err := protoyaml.Marshal(m, opts...)
				if err != nil {
					t.Fatalf("Marshal: %v", err)
				}
				fromYAML, err := protoyaml.YAMLToJSON(y)
				if err != nil {
					t.Fatalf("YAMLToJSON: %v", err)
				}
				fromProtoJSON, err := protojson.Marshal(m)
				if err != nil {
					t.Fatalf("protojson.Marshal: %v", err)
				}
				if diff := cmp.Diff(normalizeJSON(t, fromProtoJSON), normalizeJSON(t, fromYAML)); diff != "" {
					t.Errorf("semantic mismatch (-protojson +yaml):\n%s\nyaml:\n%s", diff, y)
				}
			})
		}
	}
}

// --- round-trip identity --------------------------------------------------

func TestRoundTripIdentity(t *testing.T) {
	fd := buildTestFile()
	msgs := roundTripMessages(t, fd)

	for name, m := range msgs {
		for _, flow := range []bool{false, true} {
			t.Run(name+flowSuffix(flow), func(t *testing.T) {
				var opts []protoyaml.Option
				if flow {
					opts = append(opts, protoyaml.WithFlowLeafCollections())
				}
				y, err := protoyaml.Marshal(m, opts...)
				if err != nil {
					t.Fatalf("Marshal: %v", err)
				}
				out := newLike(m)
				if err := protoyaml.Unmarshal(y, out); err != nil {
					t.Fatalf("Unmarshal: %v\nyaml:\n%s", err, y)
				}
				if !proto.Equal(m, out) {
					t.Errorf("round-trip mismatch\nyaml:\n%s\nwant: %v\ngot:  %v", y, m, out)
				}
			})
		}
	}
}

// TestRoundTripLiteralMultiline proves literal block scalars round-trip.
func TestRoundTripLiteralMultiline(t *testing.T) {
	fd := buildTestFile()
	scalarsMD := messageDescriptor(fd, "Scalars")
	m := dynamicpb.NewMessage(scalarsMD)
	query := "SELECT *\nFROM Users\nWHERE Active = TRUE"
	setField(m.ProtoReflect(), "text", protoreflect.ValueOfString(query))

	y, err := protoyaml.Marshal(m, protoyaml.WithYAMLOptions(yaml.UseLiteralStyleIfMultiline(true)))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// Sanity: the multiline string should render as a literal block.
	if !strings.Contains(string(y), "|-") {
		t.Errorf("expected literal block style, got:\n%s", y)
	}
	out := dynamicpb.NewMessage(scalarsMD)
	if err := protoyaml.Unmarshal(y, out); err != nil {
		t.Fatalf("Unmarshal: %v\nyaml:\n%s", err, y)
	}
	if !proto.Equal(m, out) {
		t.Errorf("literal round-trip mismatch\nyaml:\n%s", y)
	}
}

// --- timestamp safety -----------------------------------------------------

func TestTimestampSafety(t *testing.T) {
	ts := timestampAt(2021, 1, 2, 3, 4, 5)

	y, err := protoyaml.Marshal(ts)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// goccy quotes RFC3339 strings; prove the value survives the YAML round-trip
	// regardless of any special timestamp interpretation.
	out := &timestamppb.Timestamp{}
	if err := protoyaml.Unmarshal(y, out); err != nil {
		t.Fatalf("Unmarshal: %v\nyaml:\n%s", err, y)
	}
	if !proto.Equal(ts, out) {
		t.Errorf("timestamp round-trip mismatch\nyaml: %s\nwant: %v\ngot:  %v", y, ts, out)
	}
}

// --- YAMLToJSON unmarshal-side behavior -----------------------------------

func TestYAMLToJSON(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want string
	}{
		{"scalars", "a: 1\nb: hello\nc: true\n", `{"a":1,"b":"hello","c":true}`},
		{"nested", "outer:\n  inner: 2\n", `{"outer":{"inner":2}}`},
		{"sequence", "items:\n- 1\n- 2\n", `{"items":[1,2]}`},
		{"quotedInt64", `big: "9223372036854775807"`, `{"big":"9223372036854775807"}`},
		{"flow", "m: {k: v}\n", `{"m":{"k":"v"}}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := protoyaml.YAMLToJSON([]byte(c.yaml))
			if err != nil {
				t.Fatalf("YAMLToJSON: %v", err)
			}
			if string(got) != c.want {
				t.Errorf("YAMLToJSON = %s, want %s", got, c.want)
			}
		})
	}
}

// TestUnmarshalDiscardsUnknown confirms unknown keys are ignored (DiscardUnknown).
func TestUnmarshalDiscardsUnknown(t *testing.T) {
	fd := buildTestFile()
	scalarsMD := messageDescriptor(fd, "Scalars")
	m := dynamicpb.NewMessage(scalarsMD)
	if err := protoyaml.Unmarshal([]byte("i64: \"5\"\nunknownField: 123\n"), m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	got := m.ProtoReflect().Get(scalarsMD.Fields().ByName("i64")).Int()
	if got != 5 {
		t.Errorf("i64 = %d, want 5", got)
	}
}

// --- proto3 explicit presence ---------------------------------------------

// TestProto3OptionalPresence proves that a proto3 optional field set to its zero
// value is emitted and round-trips as present, while an unset field is omitted
// and round-trips as absent, so the two are distinguishable.
func TestProto3OptionalPresence(t *testing.T) {
	fd := buildTestFile()
	presMD := messageDescriptor(fd, "Presence")
	i32 := presMD.Fields().ByName("opt_i32")

	zero := dynamicpb.NewMessage(presMD)
	setField(zero.ProtoReflect(), "opt_i32", protoreflect.ValueOfInt32(0))
	yZero, err := protoyaml.Marshal(zero)
	if err != nil {
		t.Fatalf("Marshal(zero): %v", err)
	}
	if !strings.Contains(string(yZero), "optI32") {
		t.Errorf("set-to-zero optional field must be emitted, got:\n%s", yZero)
	}
	backZero := dynamicpb.NewMessage(presMD)
	if err := protoyaml.Unmarshal(yZero, backZero); err != nil {
		t.Fatalf("Unmarshal(zero): %v", err)
	}
	if !backZero.ProtoReflect().Has(i32) {
		t.Errorf("set-to-zero optional field must be present after round-trip")
	}

	unset := dynamicpb.NewMessage(presMD)
	yUnset, err := protoyaml.Marshal(unset)
	if err != nil {
		t.Fatalf("Marshal(unset): %v", err)
	}
	if strings.Contains(string(yUnset), "optI32") {
		t.Errorf("unset optional field must be omitted, got:\n%s", yUnset)
	}
	backUnset := dynamicpb.NewMessage(presMD)
	if err := protoyaml.Unmarshal(yUnset, backUnset); err != nil {
		t.Fatalf("Unmarshal(unset): %v", err)
	}
	if backUnset.ProtoReflect().Has(i32) {
		t.Errorf("unset optional field must be absent after round-trip")
	}
}

// --- non-string map keys --------------------------------------------------

// TestNonStringMapKeysAreJSONStrings proves int64/uint32/bool map keys render as
// JSON strings (protojson's rule) on the YAML->JSON side.
func TestNonStringMapKeysAreJSONStrings(t *testing.T) {
	fd := buildTestFile()
	mapsMD := messageDescriptor(fd, "Maps")
	m := dynamicpb.NewMessage(mapsMD)
	byI64 := m.ProtoReflect().Mutable(mapsMD.Fields().ByName("by_i64")).Map()
	byI64.Set(protoreflect.ValueOfInt64(math.MinInt64).MapKey(), protoreflect.ValueOfString("min"))
	byU32 := m.ProtoReflect().Mutable(mapsMD.Fields().ByName("by_u32")).Map()
	byU32.Set(protoreflect.ValueOfUint32(math.MaxUint32).MapKey(), protoreflect.ValueOfString("umax"))
	byBool := m.ProtoReflect().Mutable(mapsMD.Fields().ByName("by_bool")).Map()
	byBool.Set(protoreflect.ValueOfBool(true).MapKey(), protoreflect.ValueOfString("t"))
	byBool.Set(protoreflect.ValueOfBool(false).MapKey(), protoreflect.ValueOfString("f"))

	y, err := protoyaml.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	j, err := protoyaml.YAMLToJSON(y)
	if err != nil {
		t.Fatalf("YAMLToJSON: %v", err)
	}
	for _, want := range []string{
		`"-9223372036854775808":"min"`,
		`"4294967295":"umax"`,
		`"true":"t"`,
		`"false":"f"`,
	} {
		if !strings.Contains(string(j), want) {
			t.Errorf("JSON %s\nmissing string-keyed entry %s", j, want)
		}
	}
}

// --- FieldMask ------------------------------------------------------------

// TestFieldMaskForm proves a google.protobuf.FieldMask renders in protojson's
// comma-joined camelCase form and round-trips.
func TestFieldMaskForm(t *testing.T) {
	fd := buildTestFile()
	wktMD := messageDescriptor(fd, "Wkt")
	m := dynamicpb.NewMessage(wktMD)
	mask := &fieldmaskpb.FieldMask{Paths: []string{"foo_bar", "baz_qux"}}
	setField(m.ProtoReflect(), "mask", protoreflect.ValueOfMessage(mask.ProtoReflect()))

	y, err := protoyaml.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(y), "fooBar,bazQux") {
		t.Errorf("FieldMask must render camelCase comma-joined, got:\n%s", y)
	}
	back := dynamicpb.NewMessage(wktMD)
	if err := protoyaml.Unmarshal(y, back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !proto.Equal(m, back) {
		t.Errorf("FieldMask round-trip mismatch\nyaml:\n%s", y)
	}
}

// --- google.protobuf.Any --------------------------------------------------

// TestAnyDefaultResolver proves an Any wrapping a registered WKT (Duration)
// expands via the default global resolver on both Marshal and Unmarshal.
func TestAnyDefaultResolver(t *testing.T) {
	fd := buildTestFile()
	wktMD := messageDescriptor(fd, "Wkt")
	m := dynamicpb.NewMessage(wktMD)
	anyDur, err := anypb.New(durationpb.New(1500 * time.Millisecond))
	if err != nil {
		t.Fatalf("anypb.New: %v", err)
	}
	setField(m.ProtoReflect(), "payload", protoreflect.ValueOfMessage(anyDur.ProtoReflect()))

	y, err := protoyaml.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(y), "type.googleapis.com/google.protobuf.Duration") {
		t.Errorf("Any must carry its @type URL, got:\n%s", y)
	}
	back := dynamicpb.NewMessage(wktMD)
	if err := protoyaml.Unmarshal(y, back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !proto.Equal(m, back) {
		t.Errorf("Any round-trip mismatch\nyaml:\n%s", y)
	}
}

// TestAnyMarshalWithResolver exercises WithProtoJSON's explicit type Resolver on
// the marshal side: a custom registry containing only Duration expands the Any
// identically to the default global-resolver path.
func TestAnyMarshalWithResolver(t *testing.T) {
	fd := buildTestFile()
	wktMD := messageDescriptor(fd, "Wkt")
	m := dynamicpb.NewMessage(wktMD)
	anyDur, err := anypb.New(durationpb.New(1500 * time.Millisecond))
	if err != nil {
		t.Fatalf("anypb.New: %v", err)
	}
	setField(m.ProtoReflect(), "payload", protoreflect.ValueOfMessage(anyDur.ProtoReflect()))

	custom := new(protoregistry.Types)
	if err := custom.RegisterMessage(durationpb.New(0).ProtoReflect().Type()); err != nil {
		t.Fatalf("RegisterMessage: %v", err)
	}
	withResolver, err := protoyaml.Marshal(m, protoyaml.WithProtoJSON(protojson.MarshalOptions{Resolver: custom}))
	if err != nil {
		t.Fatalf("Marshal(resolver): %v", err)
	}
	def, err := protoyaml.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal(default): %v", err)
	}
	if !bytes.Equal(withResolver, def) {
		t.Errorf("explicit resolver output differs from default:\n--- resolver ---\n%s\n--- default ---\n%s", withResolver, def)
	}
}

// TestAnyUnmarshalResolverGap documents a known API gap: Marshal accepts a custom
// type Resolver via WithProtoJSON, but Unmarshal/UnmarshalJSON hardcode
// protojson.UnmarshalOptions{DiscardUnknown: true} with the default (global)
// resolver and expose no knob to supply a custom one. An Any wrapping a type
// that is known only to a custom marshal-side resolver therefore marshals
// successfully but cannot be unmarshaled. If a resolver option is ever added to
// the unmarshal side, this test should be updated to assert success.
func TestAnyUnmarshalResolverGap(t *testing.T) {
	fd := buildTestFile()
	scalarsMD := messageDescriptor(fd, "Scalars")
	wktMD := messageDescriptor(fd, "Wkt")

	// A type known only to a custom registry, not the global one.
	custom := new(protoregistry.Types)
	if err := custom.RegisterMessage(dynamicpb.NewMessageType(scalarsMD)); err != nil {
		t.Fatalf("RegisterMessage: %v", err)
	}
	inner := dynamicpb.NewMessage(scalarsMD)
	setField(inner.ProtoReflect(), "i64", protoreflect.ValueOfInt64(5))
	anyMsg, err := anypb.New(inner)
	if err != nil {
		t.Fatalf("anypb.New: %v", err)
	}
	m := dynamicpb.NewMessage(wktMD)
	setField(m.ProtoReflect(), "payload", protoreflect.ValueOfMessage(anyMsg.ProtoReflect()))

	// Marshal succeeds because the custom resolver can expand the Any.
	y, err := protoyaml.Marshal(m, protoyaml.WithProtoJSON(protojson.MarshalOptions{Resolver: custom}))
	if err != nil {
		t.Fatalf("Marshal(resolver): %v", err)
	}
	if !strings.Contains(string(y), "protoyaml_test.Scalars") {
		t.Fatalf("expected expanded Any with custom type, got:\n%s", y)
	}

	// Unmarshal cannot be given the same resolver, so it fails to resolve the
	// type URL. This asserts the current gap; do not "fix" it by hacking a
	// resolver into the fixed jsonpb options.
	back := dynamicpb.NewMessage(wktMD)
	if err := protoyaml.Unmarshal(y, back); err == nil {
		t.Errorf("expected Unmarshal to fail for a custom-only Any type (documents resolver gap); it succeeded")
	}
}

// --- flow-leaf x literal-multiline interaction ----------------------------

// TestFlowLeafLiteralMultiline proves WithFlowLeafCollections and
// yaml.UseLiteralStyleIfMultiline coexist: an all-scalar nested map becomes a
// flow mapping while a multiline string renders as a literal block, and the
// whole message still round-trips.
func TestFlowLeafLiteralMultiline(t *testing.T) {
	fd := buildTestFile()
	flMD := messageDescriptor(fd, "FlowLiteral")
	m := dynamicpb.NewMessage(flMD)
	attrs := m.ProtoReflect().Mutable(flMD.Fields().ByName("attrs")).Map()
	attrs.Set(protoreflect.ValueOfString("k1").MapKey(), protoreflect.ValueOfString("v1"))
	attrs.Set(protoreflect.ValueOfString("k2").MapKey(), protoreflect.ValueOfString("v2"))
	query := "SELECT *\nFROM Users\nWHERE Active = TRUE"
	setField(m.ProtoReflect(), "query", protoreflect.ValueOfString(query))

	y, err := protoyaml.Marshal(m,
		protoyaml.WithFlowLeafCollections(),
		protoyaml.WithYAMLOptions(yaml.UseLiteralStyleIfMultiline(true)),
	)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(y)
	if !strings.Contains(s, "attrs: {") {
		t.Errorf("expected flow-style leaf map for attrs, got:\n%s", s)
	}
	if !strings.Contains(s, "|-") {
		t.Errorf("expected literal block for multiline query, got:\n%s", s)
	}
	back := dynamicpb.NewMessage(flMD)
	if err := protoyaml.Unmarshal(y, back); err != nil {
		t.Fatalf("Unmarshal: %v\nyaml:\n%s", err, y)
	}
	if !proto.Equal(m, back) {
		t.Errorf("flow-leaf/literal round-trip mismatch\nyaml:\n%s", y)
	}
}

// --- WithProtoJSON variants -----------------------------------------------

// TestWithProtoJSONVariants proves protojson-sanctioned marshal options
// (UseProtoNames, UseEnumNumbers) flow through Marshal and stay semantically
// equal to protojson invoked with the same options.
func TestWithProtoJSONVariants(t *testing.T) {
	fd := buildTestFile()
	plan := goldenPlanNode(t, fd)

	cases := []struct {
		name string
		opts protojson.MarshalOptions
		want string // substring that proves the variant took effect
	}{
		{"UseProtoNames", protojson.MarshalOptions{UseProtoNames: true}, "display_name:"},
		{"UseEnumNumbers", protojson.MarshalOptions{UseEnumNumbers: true}, "kind: 1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			y, err := protoyaml.Marshal(plan, protoyaml.WithProtoJSON(c.opts))
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if !strings.Contains(string(y), c.want) {
				t.Errorf("variant %s did not take effect, missing %q, got:\n%s", c.name, c.want, y)
			}
			fromYAML, err := protoyaml.YAMLToJSON(y)
			if err != nil {
				t.Fatalf("YAMLToJSON: %v", err)
			}
			fromProtoJSON, err := c.opts.Marshal(plan)
			if err != nil {
				t.Fatalf("protojson.Marshal: %v", err)
			}
			if diff := cmp.Diff(normalizeJSON(t, fromProtoJSON), normalizeJSON(t, fromYAML)); diff != "" {
				t.Errorf("semantic mismatch (-protojson +yaml):\n%s\nyaml:\n%s", diff, y)
			}
		})
	}
}

// --- double lexical edges: known goccy limitations ------------------------

// TestDoubleLexicalEdgeLimitations documents (and pins) three double lexical
// forms that the goccy/go-yaml bridge cannot represent faithfully because
// goccy's YAML scalar resolver diverges from protojson (and from YAML 1.2):
//
//  1. protojson renders negative zero as "-0"; goccy decodes "-0" as the
//     integer 0, dropping the sign, so a -0 double round-trips to +0 (which
//     proto.Equal treats as unequal).
//  2. protojson renders exponent-form doubles without a decimal point
//     (1e+21, 5e-324); goccy classifies exponent-without-dot tokens as strings
//     on both decode and encode, so YAMLToJSON yields a JSON *string* rather
//     than a number. The proto round-trip still succeeds only because
//     protojson.Unmarshal leniently parses a JSON string into a double.
//
// These are reported bridge defects rooted in goccy. A minimal in-module fix is
// not possible: goccy renders a genuine string "1e+21" and a double 1e21 to the
// identical unquoted token, so there is no automatic string/number
// disambiguation, and forcing a decimal point would be a byte-incompatible
// float-canonicalization decision. This test fails loudly if goccy's behavior
// ever changes, prompting a fix and promotion of these values into the shared
// round-trip corpus.
func TestDoubleLexicalEdgeLimitations(t *testing.T) {
	fd := buildTestFile()
	edgesMD := messageDescriptor(fd, "NumEdges")
	mk := func(v float64) proto.Message {
		m := dynamicpb.NewMessage(edgesMD)
		setField(m.ProtoReflect(), "d", protoreflect.ValueOfFloat64(v))
		return m
	}

	t.Run("negativeZeroLosesSign", func(t *testing.T) {
		m := mk(math.Copysign(0, -1))
		y, err := protoyaml.Marshal(m)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if got := strings.TrimSpace(string(y)); got != "d: 0" {
			t.Errorf("negative-zero rendering changed: got %q, want \"d: 0\"; goccy may have been fixed", got)
		}
		out := dynamicpb.NewMessage(edgesMD)
		if err := protoyaml.Unmarshal(y, out); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		// KNOWN LIMITATION: the sign is lost, so the round-trip is NOT exact.
		if proto.Equal(m, out) {
			t.Errorf("negative zero now round-trips exactly; goccy limitation fixed -> add -0 to the shared corpus and drop this case")
		}
	})

	for _, tc := range []struct {
		name  string
		value float64
		token string // protojson double appears as this quoted JSON string via YAMLToJSON
	}{
		{"1e21", 1e21, `"1e+21"`},
		{"subnormal", math.SmallestNonzeroFloat64, `"5e-324"`},
	} {
		t.Run(tc.name+"RendersAsString", func(t *testing.T) {
			m := mk(tc.value)
			y, err := protoyaml.Marshal(m)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			j, err := protoyaml.YAMLToJSON(y)
			if err != nil {
				t.Fatalf("YAMLToJSON: %v", err)
			}
			// KNOWN LIMITATION: the double appears as a JSON string, not a number.
			if !strings.Contains(string(j), tc.token) {
				t.Errorf("expected goccy to render %v as JSON string %s (known limitation), got %s; goccy may have been fixed", tc.value, tc.token, j)
			}
			// The proto round-trip nonetheless succeeds via protojson leniency.
			out := dynamicpb.NewMessage(edgesMD)
			if err := protoyaml.Unmarshal(y, out); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if !proto.Equal(m, out) {
				t.Errorf("round-trip via protojson leniency unexpectedly failed\nyaml:\n%s", y)
			}
		})
	}
}

// --- helpers --------------------------------------------------------------

func flowSuffix(flow bool) string {
	if flow {
		return "/flow"
	}
	return "/block"
}

func newLike(m proto.Message) proto.Message {
	return m.ProtoReflect().New().Interface()
}

// normalizeJSON decodes JSON into a generic tree with json.Number, then
// converts every number to float64 so that lexical float differences (for
// example 1 vs 1.0) do not cause spurious mismatches while string-encoded
// int64/uint64 values remain distinct strings.
func normalizeJSON(t *testing.T, b []byte) any {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("decode json %q: %v", b, err)
	}
	return normalizeValue(v)
}

func normalizeValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = normalizeValue(val)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, val := range x {
			out[i] = normalizeValue(val)
		}
		return out
	case json.Number:
		f, err := x.Float64()
		if err != nil {
			return x.String()
		}
		return f
	default:
		return v
	}
}
