package miso

import (
	"testing"
	"time"

	"github.com/curtisnewbie/miso/util"
	"github.com/sirupsen/logrus"
)

func TestBootstrapServer(t *testing.T) {
	args := make([]string, 2)
	logrus.SetLevel(logrus.DebugLevel)

	SetProp(PropAppName, "test-app")

	go func() {
		time.Sleep(5 * time.Second)
		if IsShuttingDown() {
			t.Error()
			return
		}

		// syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		Shutdown()
	}()

	BootstrapServer(args)
}

func TestPostServerBootstrapCallback(t *testing.T) {
	i := 0

	PostServerBootstrap(func(rail Rail) error {
		i++ // 1
		if i != 1 {
			t.Fatalf("incorrect order i is %v not 1", i)
		}
		return nil
	})

	PostServerBootstrap(func(rail Rail) error {
		PostServerBootstrap(func(rail Rail) error {
			i++ // 3
			if i != 3 {
				t.Fatalf("incorrect order i is %v not 3", i)
			}
			return nil
		})
		return nil
	})

	PostServerBootstrap(func(rail Rail) error {
		i++ // 2
		if i != 2 {
			t.Fatalf("incorrect order i is %v not 2", i)
		}
		return nil
	})

	rail := EmptyRail()
	App().callPostServerBootstrapListeners(rail)

	if i != 3 {
		t.Fatalf("i is not 3, but %v", i)
	}
}

func TestPreServerBootstrapCallback(t *testing.T) {
	i := 0

	PreServerBootstrap(func(rail Rail) error {
		i++ // 1
		if i != 1 {
			t.Fatalf("incorrect order i is %v not 1", i)
		}
		return nil
	})

	PreServerBootstrap(func(rail Rail) error {
		PreServerBootstrap(func(rail Rail) error {
			i++ // 3
			if i != 3 {
				t.Fatalf("incorrect order i is %v not 3", i)
			}
			return nil
		})
		return nil
	})

	PreServerBootstrap(func(rail Rail) error {
		i++ // 2
		if i != 2 {
			t.Fatalf("incorrect order i is %v not 2", i)
		}
		return nil
	})

	rail := EmptyRail()
	App().callPreServerBootstrapListeners(rail)

	if i != 3 {
		t.Fatalf("i is not 3, but %v", i)
	}
}

func TestGroupingNestedRoutes(t *testing.T) {

	Infof("routes before: %+v", serverHttpRoutes)
	BaseRoute("/open/api").Group(

		Get("/special/order", func(inb *Inbound) (any, error) {
			// do something
			return nil, nil
		}).Extra("123", 123),

		BaseRoute("/v1").Group(
			Get("/order", func(inb *Inbound) (any, error) {
				// do something
				return nil, nil
			}).Extra("123", 123),

			Get("/shipment", func(inb *Inbound) (any, error) {
				// do something
				return nil, nil
			}),
		),

		BaseRoute("/v2").Group(
			Get("/order", func(inb *Inbound) (any, error) {
				// do something
				return nil, nil
			}).Extra("123", 123).Extra("456", 456),
			Get("/shipment", func(inb *Inbound) (any, error) {
				// do something
				return nil, nil
			}),
			Get("/invoice", func(inb *Inbound) (any, error) {
				// do something
				return nil, nil
			}),
		),
	)

	PostServerBootstrap(func(rail Rail) error {
		Info("print routes")
		for _, r := range GetHttpRoutes() {
			Infof("%+v", r)
		}

		Shutdown()
		return nil
	})

	BootstrapServer([]string{"app.name=test"})
}
func TestSetHeaderTag(t *testing.T) {
	type dummy struct {
		Name string  `header:"name"`
		Desc *string `header:"desc"`
		Age  int     `header:"age"`
	}
	var d dummy
	t.Logf("before %#v", d)

	GetHeader := func(k string) string {
		switch k {
		case "name":
			return "myname"
		case "desc":
			return "this is a test"
		case "age":
			return "???"
		}
		return ""
	}
	err := util.WalkTagShallow(&d, walkHeaderTagCallback(GetHeader))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("after %#v, Desc: %v", d, *d.Desc)
}

func BenchmarkSetHeaderTag(b *testing.B) {
	type dummy struct {
		Name string  `header:"name"`
		Desc *string `header:"desc"`
		Age  int     `header:"age"`
	}
	GetHeader := func(k string) string {
		switch k {
		case "name":
			return "myname"
		case "desc":
			return "this is a test"
		case "age":
			return "???"
		}
		return ""
	}

	d := dummy{}
	var err error
	callback := walkHeaderTagCallback(GetHeader)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = util.WalkTagShallow(&d, callback)
	}

	if err != nil {
		b.Fatal(err)
	}
}
