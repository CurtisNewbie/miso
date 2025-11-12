# JSON Processing

> [!WARNING]
>
> Since [Jsoniter](https://github.com/json-iterator/go) is nolonger actively maintained, last commit was 3 years ago. Jsoniter will be removed from this repository in later release. The default [json processing behaviour](./doc/json.md) has been removed. Previously, all struct fields are by default serialized/deserialized using lowercase camel case style without needing to add any json tag. Since v0.4.0, json tags must be added manually to maintain compatibility.
>
> Content below is deprecated.

## Default JSON Field Naming Strategy

In Golang, we export fields by capitalizing the first letter. This leads to a problem where we may have to add json tag for literally every exported fields. Miso internally maintains a custom `jsoniter` config, it configures the naming strategy that always use lowercase for the first letter of the field name unless sepcified explicitly. Whenever Miso Marshal/Unmarshal JSON values, Miso uses the configured `jsoniter` instead of the standard one. This can be reverted by modifying `json.NamingStrategyTranslate` to change the naming strategy to the one you like.

```golang
// encoding/json/json.go
var (
	config                  = jsoniter.Config{EscapeHTML: true}.Froze()

	NamingStrategyTranslate = LowercaseNamingStrategy // change this
)

type namingStrategyExtension struct {
	jsoniter.DummyExtension
}

func (extension *namingStrategyExtension) UpdateStructDescriptor(structDescriptor *jsoniter.StructDescriptor) {
	for _, binding := range structDescriptor.Fields {
		if unicode.IsLower(rune(binding.Field.Name()[0])) || binding.Field.Name()[0] == '_' {
			continue
		}
		tag, hastag := binding.Field.Tag().Lookup("json")
		if hastag {
			tagParts := strings.Split(tag, ",")
			if tagParts[0] == "-" {
				continue // hidden field
			}
			if tagParts[0] != "" {
				continue // field explicitly named
			}
		}
		binding.ToNames = []string{NamingStrategyTranslate(binding.Field.Name())}
		binding.FromNames = []string{NamingStrategyTranslate(binding.Field.Name())}
	}
}
```
