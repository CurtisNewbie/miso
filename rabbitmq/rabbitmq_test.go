package rabbitmq

import (
	"testing"

	"github.com/curtisnewbie/gocommon/common"
)

func TestInitConnection(t *testing.T) {
	common.LoadConfigFromFile("../app-conf-dev.json")
	common.SetProp(common.PROP_RABBITMQ_USERNAME, "guest")
	common.SetProp(common.PROP_RABBITMQ_PASSWORD, "guest")
	e := InitConnection()
	if e != nil {
		t.Error(e)
	}
}