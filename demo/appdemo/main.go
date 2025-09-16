package main

import (
	"os"
	"time"

	"github.com/curtisnewbie/miso/demo/api"
	"github.com/curtisnewbie/miso/middleware/mysql"
	"github.com/curtisnewbie/miso/middleware/rabbit"
	"github.com/curtisnewbie/miso/middleware/task"
	"github.com/curtisnewbie/miso/miso"
)

type DemoEvent struct {
	Value string
}

var (
	MyPipeline = rabbit.NewEventPipeline[[]DemoEvent]("demo:pipeline").
		Document("DemoPipeline", "This is a demo pipeline", "demo")
)

const (
	demoEventBusName = "event.bus.demo"
)

func init() {
	miso.SetProp("app.name", "demo")
	miso.SetProp("redis.enabled", false)           // for distributed task, set to true
	miso.SetProp("task.scheduling.enabled", false) // for distributed task, set to true
	miso.SetProp("mode.production", false)
	// miso.SetProp("server.api-doc.go.file", "goclient/client.go")
	miso.SetProp("server.api-doc.file", "api-doc.md")
}

func main() {
	// register callbacks that are invoked after configuration loaded, before server bootstrap
	miso.PreServerBootstrap(PrepareServer)

	// register callbacks that are invoked after server fully bootstrapped
	miso.PostServerBootstrap()
	// e.g.,
	// 	miso.PostServerBootstrap(TriggerWorkflowOnBootstrapped)

	// start the bootstrap process
	miso.BootstrapServer(os.Args)
}

func PrepareServer(rail miso.Rail) error {

	// declare event bus (for mq)
	rabbit.NewEventBus(demoEventBusName)

	// register some distributed tasks
	err := task.ScheduleDistributedTask(miso.Job{
		Cron:                   "*/15 * * * *",
		Name:                   "MyDistributedTask",
		LogJobExec:             true,
		TriggeredOnBoostrapped: false,
		Run: func(rail miso.Rail) error {
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
		miso.HttpPost("/open/api/demo/post", miso.AutoHandler(
			func(inb *miso.Inbound, req api.PostReq) (api.PostRes, error) {
				rail := inb.Rail()
				rail.Infof("Received request: %#v", req)

				// e.g., read some table
				var res api.PostRes
				err := mysql.GetMySQL().
					Raw(`SELECT result_id FROM post_result WHERE request_id = ?`,
						req.RequestId).
					Scan(&res).Error
				if err != nil {
					return api.PostRes{}, err
				}

				return res, nil // serialized to json
			})).
			Desc("Post demo stuff").                            // describe endpoint in api-doc
			DocHeader("Authorization", "Bearer Authorization"), // document request header
	)
	return nil
}

func TriggerWorkflowOnBootstrapped(rail miso.Rail) error {
	// maybe send some requests to other backend services
	// (using consul-based service discovery)
	var res TriggerResult
	err := miso.NewDynClient(rail, "/open/api/engine", "workflow-engine" /* service name */).
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

func doSomethingEndpoint(inb *miso.Inbound, req api.PostReq) (api.PostRes, error) {
	rail := inb.Rail()
	rail.Infof("Received request: %#v", req)
	return api.PostRes{ResultId: "1234"}, nil
}
