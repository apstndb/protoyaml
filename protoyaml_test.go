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
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
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

	return map[string]proto.Message{
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
