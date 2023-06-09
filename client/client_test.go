package client

import (
	"context"
	"testing"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/sirupsen/logrus"
)

type TestBody struct {
	Desc string `json:"description"`
}

func TestGet(t *testing.T) {
	c := common.EmptyExecContext()
	configFile := "../app-conf-dev.yml"
	common.LoadConfigFromFile(configFile, c)
	logrus.SetReportCaller(true)
	logrus.SetFormatter(common.CustomFormatter())
	common.LoadPropagationKeyProp(c)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "id", "1")
	ctx = context.WithValue(ctx, "userno", "UN123123123123")
	ctx = context.WithValue(ctx, "username", "yongj.zhuang")
	ctx = context.WithValue(ctx, "role", "admin")
	ec := common.NewExecContext(ctx, nil)

	tr := NewDefaultTClient(ec, "http://localhost:8081/open/api/test").
		AddHeaders(map[string]string{
			"TestCase": "TestGet",
		}).
		EnableTracing().
		EnableRequestLog().
		Get(map[string][]string{
			"name": {"yongj.zhuang", "zhuangyongj"},
			"age":  {"103"},
		})

	if tr.Err != nil {
		t.Fatal(tr.Err)
	}

	var body TestBody
	e := tr.ReadJson(&body)
	if e != nil {
		t.Fatal(e)
	}
	logrus.Infof("Body: %+v", body)
}

func TestDelete(t *testing.T) {
	c := common.EmptyExecContext()
	configFile := "../app-conf-dev.yml"
	common.LoadConfigFromFile(configFile, c)
	logrus.SetReportCaller(true)
	logrus.SetFormatter(common.CustomFormatter())
	common.LoadPropagationKeyProp(c)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "id", "1")
	ctx = context.WithValue(ctx, "userno", "UN123123123123")
	ctx = context.WithValue(ctx, "username", "yongj.zhuang")
	ctx = context.WithValue(ctx, "role", "admin")
	ec := common.NewExecContext(ctx, nil)

	tr := NewDefaultTClient(ec, "http://localhost:8081/open/api/test/delete").
		AddHeaders(map[string]string{
			"TestCase": "TestGet",
		}).
		EnableRequestLog().
		EnableTracing().
		Delete(map[string][]string{
			"name": {"yongj.zhuang", "zhuangyongj"},
			"age":  {"105"},
		})

	if tr.Err != nil {
		t.Fatal(tr.Err)
	}

	var body TestBody
	e := tr.ReadJson(&body)
	if e != nil {
		t.Fatal(e)
	}
	logrus.Infof("Body: %+v", body)
}

func TestPost(t *testing.T) {
	c := common.EmptyExecContext()
	configFile := "../app-conf-dev.yml"
	common.LoadConfigFromFile(configFile, c)
	logrus.SetReportCaller(true)
	logrus.SetFormatter(common.CustomFormatter())
	common.LoadPropagationKeyProp(c)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "id", "1")
	ctx = context.WithValue(ctx, "userno", "UN123123123123")
	ctx = context.WithValue(ctx, "username", "yongj.zhuang")
	ctx = context.WithValue(ctx, "role", "admin")
	ec := common.NewExecContext(ctx, nil)

	tr := NewDefaultTClient(ec, "http://localhost:8081/open/api/test/post").
		AddHeaders(map[string]string{
			"TestCase": "TestPost",
		}).
		EnableTracing().
		PostJson(TestBody{Desc: "I am the the beset"})

	if tr.Err != nil {
		t.Fatal(tr.Err)
	}

	var body TestBody
	e := tr.ReadJson(&body)
	if e != nil {
		t.Fatal(e)
	}
	logrus.Infof("Body: %+v", body)
}

func TestPut(t *testing.T) {
	c := common.EmptyExecContext()
	configFile := "../app-conf-dev.yml"
	common.LoadConfigFromFile(configFile, c)
	logrus.SetReportCaller(true)
	logrus.SetFormatter(common.CustomFormatter())
	common.LoadPropagationKeyProp(c)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "id", "1")
	ctx = context.WithValue(ctx, "userno", "UN123123123123")
	ctx = context.WithValue(ctx, "username", "yongj.zhuang")
	ctx = context.WithValue(ctx, "role", "admin")
	ec := common.NewExecContext(ctx, nil)

	tr := NewDefaultTClient(ec, "http://localhost:8081/open/api/test/put").
		AddHeaders(map[string]string{
			"TestCase": "TestPut",
		}).
		EnableTracing().
		PutJson(TestBody{Desc: "I am not the best"})

	if tr.Err != nil {
		t.Fatal(tr.Err)
	}

	var body TestBody
	e := tr.ReadJson(&body)
	if e != nil {
		t.Fatal(e)
	}
	logrus.Infof("Body: %+v", body)
}
