package miso

import (
	"testing"
	"time"

	"github.com/curtisnewbie/miso/util/rfutil"
)

func TestBootstrapServer(t *testing.T) {
	args := make([]string, 2)
	SetLogLevel("debug")

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
	App().callPostServerBoot(rail)

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
	App().callPreServerBoot(rail)

	if i != 3 {
		t.Fatalf("i is not 3, but %v", i)
	}
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
	err := rfutil.WalkTagShallow(&d, walkHeaderTagCallback(GetHeader))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("after %#v, Desc: %v", d, *d.Desc)
}
