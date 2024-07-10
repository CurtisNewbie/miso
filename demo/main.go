package main

import (
	"os"
	"time"

	"github.com/curtisnewbie/miso/middleware/mysql"
	"github.com/curtisnewbie/miso/middleware/rabbit"
	"github.com/curtisnewbie/miso/middleware/task"
	"github.com/curtisnewbie/miso/miso"
)

const (
	demoEventBusName = "event.bus.demo"
)

func init() {
	miso.SetProp("app.name", "demo")
	miso.SetProp("redis.enabled", true) // for distributed task
}

func main() {
	// register callbacks that are invoked after configuration loaded, before server bootstrap
	miso.PreServerBootstrap(PrepareServer)

	// register callbacks that are invoked after server fully bootstrapped
	miso.PostServerBootstrapped(TriggerWorkflowOnBootstrapped)

	// start the bootstrap process
	miso.BootstrapServer(os.Args)
}

func PrepareServer(rail miso.Rail) error {

	// declare event bus (for mq)
	rabbit.NewEventBus(demoEventBusName)

	// register some distributed tasks
	err := task.ScheduleDistributedTask(miso.Job{
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

	// register endpoints, api-doc are automatically generated
	// routes can also be grouped based on shared url path
	miso.BaseRoute("/open/api/demo/grouped").Group(

		// /open/api/demo/grouped/post
		miso.IPost("/open/api/demo/post",
			func(inb *miso.Inbound, req PostReq) (PostRes, error) {
				rail := inb.Rail()
				rail.Infof("Received request: %#v", req)

				// e.g., read some table
				var res PostRes
				err := mysql.GetMySQL().
					Raw(`SELECT result_id FROM post_result WHERE request_id = ?`,
						req.RequestId).
					Scan(&res).Error
				if err != nil {
					return PostRes{}, err
				}

				return res, nil // serialized to json
			}).
			Desc("Post demo stuff").                            // describe endpoint in api-doc
			DocHeader("Authorization", "Bearer Authorization"), // document request header

		miso.BaseRoute("/subgroup").Group(

			// /open/api/demo/grouped/subgroup/post
			miso.IPost("/post1", doSomethingEndpoint),
		),
	)
	return nil
}

func TriggerWorkflowOnBootstrapped(rail miso.Rail) error {
	// maybe send some requests to other backend services
	// (using consul-based service discovery)
	var res TriggerResult
	err := miso.NewDynTClient(rail, "/open/api/engine", "workflow-engine" /* service name */).
		PostJson(TriggerWorkFlow{WorkFlowId: "123"}).
		Json(&res)

	if err != nil {
		rail.Errorf("request failed, %v", err)
	} else {
		rail.Infof("request succeded, %#v", res)
	}
	return nil
}

// ----

type TriggerWorkFlow struct {
	WorkFlowId string
}
type TriggerResult struct {
	Status string
}

type PostReq struct {
	RequestId string
}
type PostRes struct {
	ResultId string
}

func doSomethingEndpoint(inb *miso.Inbound, req PostReq) (PostRes, error) {
	rail := inb.Rail()
	rail.Infof("Received request: %#v", req)
	return PostRes{ResultId: "1234"}, nil
}
