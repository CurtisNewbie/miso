package main

import (
	"os"
	"time"

	"github.com/curtisnewbie/miso/demo/api"
	"github.com/curtisnewbie/miso/middleware/mysql"
	"github.com/curtisnewbie/miso/middleware/rabbit"
	"github.com/curtisnewbie/miso/middleware/task"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"gorm.io/gorm"
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
	miso.SetProp("redis.enabled", true) // for distributed task
	miso.SetProp("mode.production", false)
	miso.SetProp("server.generate-endpoint-doc.file", "api-doc.md")
}

func main() {
	// register callbacks that are invoked after configuration loaded, before server bootstrap
	miso.PreServerBootstrap(PrepareServer)

	// register callbacks that are invoked after server fully bootstrapped
	miso.PostServerBootstrap(TriggerWorkflowOnBootstrapped)

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
	Time     util.ETime
}

func doSomethingEndpoint(inb *miso.Inbound, req PostReq) (PostRes, error) {
	rail := inb.Rail()
	rail.Infof("Received request: %#v", req)
	return PostRes{ResultId: "1234"}, nil
}

// misoapi-http: POST /api/v1
func api1(inb *miso.Inbound, req PostReq) (PostRes, error) {
	return PostRes{}, nil
}

// misoapi-http: POST /api/v2
func api2(inb *miso.Inbound, req *PostReq) (PostRes, error) {
	return PostRes{}, nil
}

// misoapi-http: POST /api/v3
func api3(inb *miso.Inbound, req *PostReq) (*PostRes, error) {
	return &PostRes{}, nil
}

// misoapi-http: POST /api/v4
func api4(inb *miso.Inbound, req api.ApiReq) (*PostRes, error) {
	return &PostRes{}, nil
}

// misoapi-http: POST /api/v5
func api5(inb *miso.Inbound, req *api.ApiReq) (*PostRes, error) {
	return &PostRes{}, nil
}

// misoapi-http: POST /api/v6
func api6(inb *miso.Inbound, req *api.ApiReq, db *gorm.DB) (*PostRes, error) {
	return &PostRes{}, nil
}

// misoapi-http: POST /api/v7
func api7(inb *miso.Inbound, req *api.ApiReq, db *gorm.DB) (api.ApiRes, error) {
	return api.ApiRes{}, nil
}

// misoapi-http: POST /api/v8
func api8(inb *miso.Inbound, req *api.ApiReq, db *gorm.DB) (*api.ApiRes, error) {
	return &api.ApiRes{}, nil
}

// misoapi-http: POST /api/v9
func api9(inb *miso.Inbound, req *api.ApiReq, db *gorm.DB) (*[]api.ApiRes, error) {
	return nil, nil
}

// misoapi-http: POST /api/v10
func api10(inb *miso.Inbound, req *api.ApiReq, db *gorm.DB) ([]api.ApiRes, error) {
	return nil, nil
}

// misoapi-http: POST /api/v11
func api11(inb *miso.Inbound, req *api.ApiReq, db *gorm.DB) ([]PostRes, error) {
	return nil, nil
}

// misoapi-http: POST /api/v12
func api12(inb *miso.Inbound, req []api.ApiReq, db *gorm.DB) ([]PostRes, error) {
	return nil, nil
}

// misoapi-http: POST /api/v13
func api13(inb *miso.Inbound, req []api.ApiReq, db *gorm.DB) (any, error) {
	return nil, nil
}

// misoapi-http: POST /api/v14
func api14(inb *miso.Inbound, req api.ApiReq, db *gorm.DB) ([]PostRes, error) {
	return nil, nil
}

// misoapi-http: GET /api/v15
func api15(inb *miso.Inbound, db *gorm.DB) ([]PostRes, error) {
	return nil, nil
}

// misoapi-http: GET /api/v16
// misoapi-ngtable
func api16(inb *miso.Inbound, db *gorm.DB) (miso.PageRes[PostRes], error) {
	return miso.PageRes[PostRes]{}, nil
}

// misoapi-http: GET /api/v17
func api17(inb *miso.Inbound, db *gorm.DB) []PostRes {
	return []PostRes{}
}

// misoapi-http: POST /api/v18
func api18(inb *miso.Inbound, db *gorm.DB) {
}

// misoapi-http: GET /api/v19
func api19(inb *miso.Inbound, db *gorm.DB) error {
	return nil
}
