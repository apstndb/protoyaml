// Package protoyaml renders and parses protobuf messages as YAML using the
// canonical protojson mapping.
//
// # Definition
//
// protobuf's JSON representation is defined by protojson. This package renders
// and parses that representation in YAML syntax, using goccy/go-yaml as the
// YAML engine.
//
// By default the output is the canonical protojson mapping. protojson-sanctioned
// variants are opt-in through WithProtoJSON: passing a protojson.MarshalOptions
// value enables the knobs protojson itself defines (UseProtoNames,
// UseEnumNumbers, EmitUnpopulated, a custom type Resolver, and so on). There is
// no non-protojson (reflection-based) mode: every value this package emits or
// accepts is a protojson representation written in YAML rather than JSON syntax.
//
// This is unrelated to github.com/bufbuild/protoyaml-go, which uses a different
// YAML engine and pursues different design goals.
//
// # Marshal pipeline
//
// Marshal drives protojson for semantics and goccy/go-yaml for syntax:
//
//  1. protojson.Marshal(m) produces canonical JSON.
//  2. The JSON is decoded with goccy/go-yaml using UseOrderedMap. Because JSON
//     is valid YAML flow syntax, this yields an ordered value tree that
//     preserves protojson's key order (field-number order, map keys sorted by
//     protojson).
//  3. The ordered value is rendered as YAML, optionally with flow-style leaf
//     collections (see WithFlowLeafCollections).
//
// # Unmarshal pipeline
//
// Unmarshal converts YAML to JSON with YAMLToJSON and then decodes with
// protojson using DiscardUnknown, so unknown keys are ignored rather than
// rejected. UnmarshalJSON exposes the JSON-side decode directly.
//
// # Compatibility
//
// The exact output bytes are part of the compatibility surface: a change to the
// rendered bytes is a breaking change. The semantics are inherited from
// protojson, so protobuf's JSON mapping rules (enum names, int64 as string,
// well-known type encodings, and so on) apply unchanged.
package protoyaml
