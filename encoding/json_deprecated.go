package encoding

import (
	"github.com/curtisnewbie/miso/encoding/json"
)

// Deprecated: use encoding/json package instead, this is only for compatibility.
var (
	LowercaseNamingStrategy = json.LowercaseNamingStrategy
	ParseJson               = json.ParseJson
	SParseJson              = json.SParseJson
	WriteJson               = json.WriteJson
	SWriteJson              = json.SWriteJson
	CustomSWriteJson        = json.CustomSWriteJson
	DecodeJson              = json.DecodeJson
	EncodeJson              = json.EncodeJson
)
