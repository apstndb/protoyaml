package protoyaml

import (
	"encoding/json"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// config holds the resolved options for a Marshal call.
type config struct {
	yamlOpts  []yaml.EncodeOption
	protojson protojson.MarshalOptions
	flowLeaf  bool
}

// Option configures Marshal. Options are applied in order.
type Option func(*config)

// WithYAMLOptions passes goccy/go-yaml encode options through to the YAML
// rendering stage. Use it for stylistic control such as yaml.Indent,
// yaml.IndentSequence, or yaml.UseLiteralStyleIfMultiline. These options only
// affect the lexical shape of the output; they never change its semantics.
func WithYAMLOptions(opts ...yaml.EncodeOption) Option {
	return func(c *config) {
		c.yamlOpts = append(c.yamlOpts, opts...)
	}
}

// WithFlowLeafCollections renders every mapping whose values are all scalars in
// YAML flow style (for example {k: v, k2: v2}) instead of block style.
// Sequences always stay in block style, non-leaf mappings (those that contain
// a nested mapping or sequence) always stay in block style, and the document
// root mapping always stays in block style so the output reads as a YAML
// document even when the whole message is scalar-only. The result is more
// compact for deeply nested leaf records while keeping the outer structure
// readable.
func WithFlowLeafCollections() Option {
	return func(c *config) {
		c.flowLeaf = true
	}
}

// WithProtoJSON sets the protojson.MarshalOptions used for the protojson stage
// of Marshal. The zero value (the canonical protojson mapping) is used by
// default; passing a non-zero value opts into the protojson-sanctioned variants
// protojson itself defines (EmitUnpopulated, UseProtoNames, UseEnumNumbers, a
// custom type Resolver, and so on). Marshal always drives protojson for its
// semantics; this option only selects which protojson marshal options apply.
//
// This configures the marshal side only. The unmarshal side uses a fixed
// protojson.UnmarshalOptions{DiscardUnknown: true} with the default (global)
// type resolver; there is currently no option to supply a custom resolver to
// Unmarshal (see UnmarshalJSON).
func WithProtoJSON(o protojson.MarshalOptions) Option {
	return func(c *config) {
		c.protojson = o
	}
}

// jsonpb is the fixed protojson decode configuration for the unmarshal side.
// DiscardUnknown mirrors the behavior of the reference implementation in
// github.com/apstndb/spannerplan so that YAML documents carrying extra keys do
// not fail to load.
var jsonpb = protojson.UnmarshalOptions{
	DiscardUnknown: true,
}

// Marshal renders m as YAML using the protojson mapping. By default that is the
// canonical protojson mapping; WithProtoJSON opts into the protojson-sanctioned
// variants (UseProtoNames, UseEnumNumbers, EmitUnpopulated, a custom Resolver).
//
// The pipeline is: protojson.Marshal(m) produces the JSON, that JSON is parsed
// by goccy/go-yaml with UseOrderedMap so protojson's key order is preserved
// (JSON is valid YAML flow syntax), and the ordered value is rendered back out
// as YAML. There is intentionally no non-protojson (reflection) path.
func Marshal(m proto.Message, opts ...Option) ([]byte, error) {
	var cfg config
	for _, o := range opts {
		o(&cfg)
	}

	// protojson is the semantics anchor: whatever it emits is, by definition,
	// the canonical representation this package renders in YAML syntax.
	j, err := cfg.protojson.Marshal(m)
	if err != nil {
		return nil, err
	}

	// Lossless ordered bridge: JSON is valid YAML flow syntax, so decoding with
	// UseOrderedMap yields yaml.MapSlice trees that preserve protojson's field
	// order (field-number order, with map keys sorted by protojson).
	var v any
	if err := yaml.UnmarshalWithOptions(j, &v, yaml.UseOrderedMap()); err != nil {
		return nil, err
	}

	if !cfg.flowLeaf {
		return yaml.MarshalWithOptions(v, cfg.yamlOpts...)
	}

	// For flow-leaf rendering we build the encoder's AST (which carries correct
	// positions for inline flow rendering) and flip qualifying mappings to flow
	// style before serializing. Re-parsing block output and flipping does not
	// re-render inline correctly, so we go through ValueToNode instead.
	node, err := yaml.ValueToNode(v, cfg.yamlOpts...)
	if err != nil {
		return nil, err
	}
	walkFlowLeaf(node)
	out := node.String()
	// node.String() does not append a trailing newline; yaml.Marshal does, so
	// normalize to match and keep the two paths consistent.
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return []byte(out), nil
}

// walkFlowLeaf applies the flow-leaf transform below the document root: the
// root mapping itself always stays block so the output reads as a YAML
// document, while every nested leaf mapping is flipped to flow style.
func walkFlowLeaf(root ast.Node) {
	if m, ok := root.(*ast.MappingNode); ok {
		for _, val := range m.Values {
			ast.Walk(flowLeafVisitor{}, val.Value)
		}
		return
	}
	ast.Walk(flowLeafVisitor{}, root)
}

// flowLeafVisitor flips mappings whose values are all scalars to flow style.
type flowLeafVisitor struct{}

// Visit implements ast.Visitor. It returns itself so ast.Walk keeps descending.
func (flowLeafVisitor) Visit(n ast.Node) ast.Visitor {
	m, ok := n.(*ast.MappingNode)
	if !ok {
		return flowLeafVisitor{}
	}
	for _, val := range m.Values {
		switch val.Value.(type) {
		case *ast.MappingNode, *ast.SequenceNode:
			// A nested collection makes this a non-leaf mapping: leave it block.
			return flowLeafVisitor{}
		}
	}
	m.SetIsFlowStyle(true)
	return flowLeafVisitor{}
}

// Unmarshal parses YAML into m using the canonical protojson mapping.
//
// It converts YAML to JSON (see YAMLToJSON) and then decodes with
// protojson using DiscardUnknown, so unknown fields are ignored rather than
// rejected.
func Unmarshal(b []byte, m proto.Message) error {
	j, err := YAMLToJSON(b)
	if err != nil {
		return err
	}
	return UnmarshalJSON(j, m)
}

// UnmarshalJSON decodes canonical protojson bytes into m with DiscardUnknown.
func UnmarshalJSON(j []byte, m proto.Message) error {
	return jsonpb.Unmarshal(j, m)
}

// YAMLToJSON converts YAML bytes into JSON bytes suitable for
// protojson.Unmarshal. It decodes the YAML into a generic value with
// goccy/go-yaml (which normalizes mapping keys to strings) and re-encodes it
// with encoding/json. This mirrors the reference implementation in
// github.com/apstndb/spannerplan.
func YAMLToJSON(y []byte) ([]byte, error) {
	var i any
	if err := yaml.Unmarshal(y, &i); err != nil {
		return nil, err
	}
	return json.Marshal(i)
}
