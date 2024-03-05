package miso

import (
	"testing"
	"time"
)

func PreTest(t *testing.T) Rail {
	rail := EmptyRail()
	err := LoadConfigFromFile("../testdata/conf_dev.yml", rail)
	if err != nil {
		t.Fatal(err)
	}
	SetLogLevel("debug")
	err = InitConsulClient()
	if err != nil {
		t.Fatal(err)
	}
	return rail
}

func TestPollInstances(t *testing.T) {
	rail := PreTest(t)
	sl := GetServerList()
	if sl == nil {
		t.Fatal("sl is nil")
	}
	if err := sl.PollInstances(rail); err != nil {
		t.Fatal(err)
	}

	servers := sl.ListServers(rail, "vfm")
	if len(servers) < 1 {
		t.Fatal("servers is empty")
	}
	t.Logf("%#v", servers)
}

func TestPollInstance(t *testing.T) {
	rail := PreTest(t)
	sl := GetServerList()
	if sl == nil {
		t.Fatal("sl is nil")
	}
	if err := sl.PollInstance(rail, "vfm"); err != nil {
		t.Fatal(err)
	}

	servers := sl.ListServers(rail, "vfm")
	if len(servers) < 1 {
		t.Fatal("servers is empty")
	}
	t.Logf("%#v", servers)
}

func TestSubscribe(t *testing.T) {
	rail := PreTest(t)
	sl := GetServerList()
	if sl == nil {
		t.Fatal("sl is nil")
	}
	if err := sl.Subscribe(rail, "vfm"); err != nil {
		t.Fatal(err)
	}
	if !sl.IsSubscribed(rail, "vfm") {
		t.Fatal("vfm not subscribed")
	}

	time.Sleep(time.Second * 1)

	servers := sl.ListServers(rail, "vfm")
	if len(servers) < 1 {
		t.Fatal("servers is empty")
	}
	t.Logf("%#v", servers)
}

func TestUnsubscribeAll(t *testing.T) {
	rail := PreTest(t)
	sl := GetServerList()
	if sl == nil {
		t.Fatal("sl is nil")
	}
	if err := sl.Subscribe(rail, "vfm"); err != nil {
		t.Fatal(err)
	}

	if err := sl.UnsubscribeAll(rail); err != nil {
		t.Fatal(err)
	}
}

func TestUnsubscribe(t *testing.T) {
	rail := PreTest(t)
	sl := GetServerList()
	if sl == nil {
		t.Fatal("sl is nil")
	}
	if err := sl.Subscribe(rail, "vfm"); err != nil {
		t.Fatal(err)
	}

	if err := sl.Unsubscribe(rail, "vfm"); err != nil {
		t.Fatal(err)
	}
}
