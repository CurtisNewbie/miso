# API Endpoints

## Contents

- [POST /open/api/demo/grouped/open/api/demo/post](#post-openapidemogroupedopenapidemopost)
- [POST /open/api/demo/grouped/subgroup/post1](#post-openapidemogroupedsubgrouppost1)
- [GET /metrics](#get-metrics)
- [GET /debug/pprof](#get-debugpprof)
- [GET /debug/pprof/:name](#get-debugpprofname)
- [GET /debug/pprof/cmdline](#get-debugpprofcmdline)
- [GET /debug/pprof/profile](#get-debugpprofprofile)
- [GET /debug/pprof/symbol](#get-debugpprofsymbol)
- [GET /debug/pprof/trace](#get-debugpproftrace)
- [GET /doc/api](#get-docapi)

## POST /open/api/demo/grouped/open/api/demo/post

- Description: Post demo stuff
- Header Parameter:
  - "Authorization": Bearer Authorization
- JSON Request:
    - "requestId": (string) 
- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/open/api/demo/grouped/open/api/demo/post' \
    -H 'Authorization: ' \
    -H 'Content-Type: application/json' \
    -d '{"requestId":""}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type PostReq struct {
  	RequestId string
  }

  type PostRes struct {
  	ResultId string
  	Time util.ETime
  }

  func SendPostReq(rail miso.Rail, req PostReq, authorization string) (PostRes, error) {
  	var res miso.GnResp[PostRes]
  	err := miso.NewDynTClient(rail, "/open/api/demo/grouped/open/api/demo/post", "demo").
  		AddHeader("authorization", authorization).
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  		var dat PostRes
  		return dat, err
  	}
  	dat, err := res.Res()
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  	}
  	return dat, err
  }
  ```

- JSON Request Object In TypeScript:
  ```ts
  export interface PostReq {
    requestId?: string;
  }
  ```

- JSON Response Object In TypeScript:
  ```ts
  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
    data?: PostRes;
  }

  export interface PostRes {
    resultId?: string;
    time?: number;
  }
  ```

- Angular HttpClient Demo:
  ```ts
  import { MatSnackBar } from "@angular/material/snack-bar";
  import { HttpClient } from "@angular/common/http";

  constructor(
    private snackBar: MatSnackBar,
    private http: HttpClient
  ) {}

  sendPostReq() {
    let authorization: any | null = null;
    let req: PostReq | null = null;
    this.http.post<any>(`/demo/open/api/demo/grouped/open/api/demo/post`, req,
      {
        headers: {
          "Authorization": authorization
        }
      })
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: PostRes = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## POST /open/api/demo/grouped/subgroup/post1

- JSON Request:
    - "requestId": (string) 
- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/open/api/demo/grouped/subgroup/post1' \
    -H 'Content-Type: application/json' \
    -d '{"requestId":""}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type PostReq struct {
  	RequestId string
  }

  type PostRes struct {
  	ResultId string
  	Time util.ETime
  }

  func SendPostReq(rail miso.Rail, req PostReq) (PostRes, error) {
  	var res miso.GnResp[PostRes]
  	err := miso.NewDynTClient(rail, "/open/api/demo/grouped/subgroup/post1", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  		var dat PostRes
  		return dat, err
  	}
  	dat, err := res.Res()
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  	}
  	return dat, err
  }
  ```

- JSON Request Object In TypeScript:
  ```ts
  export interface PostReq {
    requestId?: string;
  }
  ```

- JSON Response Object In TypeScript:
  ```ts
  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
    data?: PostRes;
  }

  export interface PostRes {
    resultId?: string;
    time?: number;
  }
  ```

- Angular HttpClient Demo:
  ```ts
  import { MatSnackBar } from "@angular/material/snack-bar";
  import { HttpClient } from "@angular/common/http";

  constructor(
    private snackBar: MatSnackBar,
    private http: HttpClient
  ) {}

  sendPostReq() {
    let req: PostReq | null = null;
    this.http.post<any>(`/demo/open/api/demo/grouped/subgroup/post1`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: PostRes = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## GET /metrics

- Description: Collect prometheus metrics information
- Header Parameter:
  - "Authorization": Basic authorization if enabled
- cURL:
  ```sh
  curl -X GET 'http://localhost:8080/metrics' \
    -H 'Authorization: '
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  func SendRequest(rail miso.Rail, authorization string) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynTClient(rail, "/metrics", "demo").
  		AddHeader("authorization", authorization).
  		Get().
  		Json(&res)
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  		return err
  	}
  	err = res.Err()
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  	}
  	return err
  }
  ```

- Angular HttpClient Demo:
  ```ts
  import { MatSnackBar } from "@angular/material/snack-bar";
  import { HttpClient } from "@angular/common/http";

  constructor(
    private snackBar: MatSnackBar,
    private http: HttpClient
  ) {}

  sendRequest() {
    let authorization: any | null = null;
    this.http.get<any>(`/demo/metrics`,
      {
        headers: {
          "Authorization": authorization
        }
      })
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## GET /debug/pprof

- cURL:
  ```sh
  curl -X GET 'http://localhost:8080/debug/pprof'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  func SendRequest(rail miso.Rail) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynTClient(rail, "/debug/pprof", "demo").
  		Get().
  		Json(&res)
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  		return err
  	}
  	err = res.Err()
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  	}
  	return err
  }
  ```

- Angular HttpClient Demo:
  ```ts
  import { MatSnackBar } from "@angular/material/snack-bar";
  import { HttpClient } from "@angular/common/http";

  constructor(
    private snackBar: MatSnackBar,
    private http: HttpClient
  ) {}

  sendRequest() {
    this.http.get<any>(`/demo/debug/pprof`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## GET /debug/pprof/:name

- cURL:
  ```sh
  curl -X GET 'http://localhost:8080/debug/pprof/:name'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  func SendRequest(rail miso.Rail) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynTClient(rail, "/debug/pprof/:name", "demo").
  		Get().
  		Json(&res)
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  		return err
  	}
  	err = res.Err()
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  	}
  	return err
  }
  ```

- Angular HttpClient Demo:
  ```ts
  import { MatSnackBar } from "@angular/material/snack-bar";
  import { HttpClient } from "@angular/common/http";

  constructor(
    private snackBar: MatSnackBar,
    private http: HttpClient
  ) {}

  sendRequest() {
    this.http.get<any>(`/demo/debug/pprof/:name`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## GET /debug/pprof/cmdline

- cURL:
  ```sh
  curl -X GET 'http://localhost:8080/debug/pprof/cmdline'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  func SendRequest(rail miso.Rail) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynTClient(rail, "/debug/pprof/cmdline", "demo").
  		Get().
  		Json(&res)
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  		return err
  	}
  	err = res.Err()
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  	}
  	return err
  }
  ```

- Angular HttpClient Demo:
  ```ts
  import { MatSnackBar } from "@angular/material/snack-bar";
  import { HttpClient } from "@angular/common/http";

  constructor(
    private snackBar: MatSnackBar,
    private http: HttpClient
  ) {}

  sendRequest() {
    this.http.get<any>(`/demo/debug/pprof/cmdline`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## GET /debug/pprof/profile

- cURL:
  ```sh
  curl -X GET 'http://localhost:8080/debug/pprof/profile'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  func SendRequest(rail miso.Rail) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynTClient(rail, "/debug/pprof/profile", "demo").
  		Get().
  		Json(&res)
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  		return err
  	}
  	err = res.Err()
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  	}
  	return err
  }
  ```

- Angular HttpClient Demo:
  ```ts
  import { MatSnackBar } from "@angular/material/snack-bar";
  import { HttpClient } from "@angular/common/http";

  constructor(
    private snackBar: MatSnackBar,
    private http: HttpClient
  ) {}

  sendRequest() {
    this.http.get<any>(`/demo/debug/pprof/profile`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## GET /debug/pprof/symbol

- cURL:
  ```sh
  curl -X GET 'http://localhost:8080/debug/pprof/symbol'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  func SendRequest(rail miso.Rail) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynTClient(rail, "/debug/pprof/symbol", "demo").
  		Get().
  		Json(&res)
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  		return err
  	}
  	err = res.Err()
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  	}
  	return err
  }
  ```

- Angular HttpClient Demo:
  ```ts
  import { MatSnackBar } from "@angular/material/snack-bar";
  import { HttpClient } from "@angular/common/http";

  constructor(
    private snackBar: MatSnackBar,
    private http: HttpClient
  ) {}

  sendRequest() {
    this.http.get<any>(`/demo/debug/pprof/symbol`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## GET /debug/pprof/trace

- cURL:
  ```sh
  curl -X GET 'http://localhost:8080/debug/pprof/trace'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  func SendRequest(rail miso.Rail) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynTClient(rail, "/debug/pprof/trace", "demo").
  		Get().
  		Json(&res)
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  		return err
  	}
  	err = res.Err()
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  	}
  	return err
  }
  ```

- Angular HttpClient Demo:
  ```ts
  import { MatSnackBar } from "@angular/material/snack-bar";
  import { HttpClient } from "@angular/common/http";

  constructor(
    private snackBar: MatSnackBar,
    private http: HttpClient
  ) {}

  sendRequest() {
    this.http.get<any>(`/demo/debug/pprof/trace`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## GET /doc/api

- Description: Serve the generated API documentation webpage
- Expected Access Scope: PUBLIC
- cURL:
  ```sh
  curl -X GET 'http://localhost:8080/doc/api'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  func SendRequest(rail miso.Rail) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynTClient(rail, "/doc/api", "demo").
  		Get().
  		Json(&res)
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  		return err
  	}
  	err = res.Err()
  	if err != nil {
  		rail.Errorf("Request failed, %v", err)
  	}
  	return err
  }
  ```

- Angular HttpClient Demo:
  ```ts
  import { MatSnackBar } from "@angular/material/snack-bar";
  import { HttpClient } from "@angular/common/http";

  constructor(
    private snackBar: MatSnackBar,
    private http: HttpClient
  ) {}

  sendRequest() {
    this.http.get<any>(`/demo/doc/api`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

# Event Pipelines

- DemoPipeline
  - Description: This is a demo pipeline
  - RabbitMQ Queue: `demo:pipeline`
  - RabbitMQ Exchange: `demo:pipeline`
  - RabbitMQ RoutingKey: `#`
  - Event Payload: (array)
    - "value": (string) 
