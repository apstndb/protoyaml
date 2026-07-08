# protoyaml

`protoyaml` is a [goccy/go-yaml](https://github.com/goccy/go-yaml)-based canonical protojson&hArr;YAML bridge. It is **unrelated to [bufbuild/protoyaml-go](https://github.com/bufbuild/protoyaml-go)**, which is built on a different YAML engine and pursues different design goals.

## Definition

protobuf's JSON representation is defined by [`protojson`](https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson). This module renders and parses that representation in YAML syntax.

**By default the output is the canonical protojson mapping.** protojson-sanctioned variants are opt-in through [`WithProtoJSON`](#api): passing a `protojson.MarshalOptions` value enables the knobs protojson itself defines — `UseProtoNames`, `UseEnumNumbers`, `EmitUnpopulated`, a custom type `Resolver`, and so on. There is **no non-protojson (reflection-based) mode**: every value this module emits or accepts is a protojson representation written in YAML rather than JSON syntax.

Why anchor on protojson? The protojson mapping already answers every representation question protobuf has (enum names, `int64` as string, well-known type encodings such as `Timestamp`/`Duration`/`Struct`, and so on). Rendering that mapping in YAML syntax keeps the behavior predictable and the semantics identical to protojson; a second, YAML-specific mapping would only introduce ambiguity. So `protoyaml` treats protojson as the semantics anchor and uses goccy/go-yaml purely for syntax.

## API

```go
func Marshal(m proto.Message, opts ...Option) ([]byte, error)
func Unmarshal(b []byte, m proto.Message) error      // YAML -> JSON -> protojson.Unmarshal (DiscardUnknown)
func UnmarshalJSON(j []byte, m proto.Message) error  // protojson.Unmarshal (DiscardUnknown)
func YAMLToJSON(y []byte) ([]byte, error)             // goccy YAML -> interface{} -> encoding/json

func WithYAMLOptions(opts ...yaml.EncodeOption) Option // style pass-through
func WithFlowLeafCollections() Option                 // see below
func WithProtoJSON(o protojson.MarshalOptions) Option // default: zero value
```

### Marshal pipeline

1. `protojson.Marshal(m)` produces canonical JSON (the semantics anchor).
2. That JSON is decoded with goccy/go-yaml using `UseOrderedMap`. Because JSON is valid YAML flow syntax, this yields an ordered value tree that preserves protojson's key order (field-number order, map keys sorted by protojson).
3. The ordered value is rendered as YAML, optionally reshaped by `WithFlowLeafCollections`.

### Example

```go
out, _ := protoyaml.Marshal(planNode, protoyaml.WithFlowLeafCollections())
fmt.Print(string(out))
```

```yaml
index: 1
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
```

### WithFlowLeafCollections

By default every collection renders in block style. `WithFlowLeafCollections` renders every **mapping whose values are all scalars** in flow style (`{k: v, k2: v2}`) instead. The rules are:

- **Leaf mappings** (all values are scalars) become flow style.
- **Non-leaf mappings** (containing a nested mapping or sequence) stay block style.
- **Sequences** always stay block style.

This keeps the outer structure readable while compacting the innermost records. It is implemented by building goccy's encoder AST and flipping the flow-style flag on qualifying `MappingNode`s before serialization, so the result is real YAML produced by goccy, not string surgery.

## Compatibility

The exact output bytes are part of the compatibility surface: **a change to the rendered bytes is a breaking change**, subject to this module's versioning policy. The semantics are inherited from protojson, so protobuf's JSON mapping rules apply unchanged; changes in the protobuf library's protojson output propagate here.

## Known limitations

These stem from goccy/go-yaml's scalar handling and are pinned by characterization tests (`TestDoubleLexicalEdgeLimitations`, `TestAnyUnmarshalResolverGap`) so any behavior change is caught:

- **Negative zero**: protojson emits `-0` for a negative-zero double, but the YAML bridge decodes it as integer `0`, so the sign is lost on Marshal and the value round-trips to `+0`.
- **Exponent-form doubles without a decimal point** (e.g. `1e+21`, `5e-324`): the emitted YAML scalar is unquoted and reads back as a string in the generic JSON tree (`YAMLToJSON`). `Unmarshal` into a proto message still works because protojson accepts string-encoded numbers for double fields.
- **`Any` with a custom type resolver**: `WithProtoJSON` supplies a resolver to Marshal only; `Unmarshal`/`UnmarshalJSON` use the global type registry. An `Any` whose type is known only to a custom resolver marshals but does not unmarshal. An unmarshal-side option may be added later.

## License

MIT. See [LICENSE](LICENSE).
