# JSON Processing

## Default JSON Field Naming Strategy

In Golang, we export fields by capitalizing the first letter. This leads to a problem where we may have to add json tag for literally every exported fields. Miso internally uses `jsoniter`, it configures the naming strategy that always use lowercase for the first letter of the field name unless sepcified explicitly. Whenever Miso Marshal/Unmarshal JSON values, Miso uses the configured `jsoniter` instead of the standard one. This can be reverted by registering `PreServerBootstrap` callback to change the naming strategy back to the default one.