package main

import (
	"os"
	"time"

	"github.com/curtisnewbie/miso/miso"
	"github.com/gin-gonic/gin"
)

const (
	demoEventBusName = "event.bus.demo"
)

func init() {
	miso.SetProp("app.name", "demo")
	miso.SetProp("redis.enabled", true) // for distributed task
}

func main() {

	// after configuration loaded, before server bootstrap
	miso.PreServerBootstrap(func(rail miso.Rail) error {

		// declare event bus (for mq)
		miso.NewEventBus(demoEventBusName)

		// register some distributed tasks
		err := miso.ScheduleDistributedTask(miso.Job{
			Cron:                   "*/15 * * * *",
			CronWithSeconds:        false,
			Name:                   "MyDistributedTask",
			LogJobExec:             true,
			TriggeredOnBoostrapped: false,
			Run: func(miso miso.Rail) error {
				rail.Infof("MyDistributedTask running, now: %v", time.Now())
				return nil
			},
		})
		if err != nil {
			panic(err) // for demo only
		}

		type MyReq struct {
			Name string
			Age  int
		}
		type MyRes struct {
			ResultId string
		}
		// register endpoints, api-doc are automatically  generated based on MyReq/MyRes
		miso.IPost("/open/api/demo/post",
			func(c *gin.Context, rail miso.Rail, req MyReq) (MyRes, error) {
				rail.Infof("Received request: %#v", req)
				return MyRes{ResultId: "1234"}, nil // serialized to json
			}).
			Desc("Post demo stuff").                           // add description to api-doc
			DocHeader("Authorization", "Bearer Authorization") // document header param in api-doc

		// group routes based on shared url path
		miso.BaseRoute("/open/api/demo/grouped").Group(

			// /open/api/demo/grouped/post1
			miso.IPost("/post1", doSomethingEndpoint),

			miso.BaseRoute("/subgroup").Group(

				// /open/api/demo/grouped/subgroup/post2
				miso.IPost("/post2", doSomethingEndpoint),
			),
		)

		return nil
	})

	// after server fully bootstrapped, do some stuff
	miso.PostServerBootstrapped(func(rail miso.Rail) error {

		type GetUserInfoReq struct {
			UserNo string
		}
		type GetUserInfoRes struct {
			Username     string
			RegisteredAt miso.ETime
		}

		// maybe send some requests to other backend services
		// (using consul-based service discovery)
		var res GetUserInfoRes
		err := miso.NewDynTClient(rail, "/open/api/user", "user-vault").
			PostJson(GetUserInfoReq{UserNo: "123"}).
			Json(&res)

		if err != nil {
			rail.Errorf("request failed, %v", err)
		} else {
			rail.Infof("request succeded, %#v", res)
		}

		return nil
	})

	// bootstrap server
	miso.BootstrapServer(os.Args)
}

type MyReq struct {
	Name string
	Age  int
}
type MyRes struct {
	ResultId string
}

func doSomethingEndpoint(c *gin.Context, rail miso.Rail, req MyReq) (MyRes, error) {
	rail.Infof("Received request: %#v", req)
	return MyRes{ResultId: "1234"}, nil
}
