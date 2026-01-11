# API Endpoints

## Contents

- [POST /api/v1](#post-apiv1)
- [POST /api/v2](#post-apiv2)
- [POST /api/v3](#post-apiv3)
- [POST /api/v4](#post-apiv4)
- [POST /api/v5](#post-apiv5)
- [POST /api/v6](#post-apiv6)
- [POST /api/v7](#post-apiv7)
- [POST /api/v8](#post-apiv8)
- [POST /api/v9](#post-apiv9)
- [POST /api/v10](#post-apiv10)
- [POST /api/v11](#post-apiv11)
- [POST /api/v12](#post-apiv12)
- [POST /api/v13](#post-apiv13)
- [POST /api/v14](#post-apiv14)
- [GET /api/v15](#get-apiv15)
- [GET /api/v16](#get-apiv16)
- [GET /api/v17](#get-apiv17)
- [POST /api/v18](#post-apiv18)
- [GET /api/v19](#get-apiv19)
- [POST /api/v20](#post-apiv20)
- [POST /api/v21](#post-apiv21)
- [POST /api/v22](#post-apiv22)
- [POST /api/v23](#post-apiv23)
- [POST /api/v24](#post-apiv24)
- [POST /api/v25](#post-apiv25)
- [OPTIONS /api/v26](#options-apiv26)
- [HEAD /api/v27](#head-apiv27)
- [PATCH /api/v28](#patch-apiv28)
- [CONNECT /api/v29](#connect-apiv29)
- [TRACE /api/v30](#trace-apiv30)
- [POST /api/v31](#post-apiv31)
- [POST /api/v32](#post-apiv32)
- [POST /api/v33](#post-apiv33)
- [POST /open/api/demo/grouped/open/api/demo/post](#post-openapidemogroupedopenapidemopost)
- [GET /debug/trace/recorder/run](#get-debugtracerecorderrun)
- [GET /debug/trace/recorder/snapshot](#get-debugtracerecordersnapshot)
- [GET /debug/trace/recorder/stop](#get-debugtracerecorderstop)
- [POST /debug/task/disable-workers](#post-debugtaskdisable-workers)
- [POST /debug/task/enable-workers](#post-debugtaskenable-workers)

## POST /api/v1

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
  curl -X POST 'http://localhost:8080/api/v1' \
    -H 'Content-Type: application/json' \
    -d '{"requestId":""}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type PostReq struct {
  	RequestId string `json:"requestId"`
  }

  type PostRes struct {
  	ResultId string `json:"resultId"`
  	Time atom.Time `json:"time"`
  }

  func api1(rail miso.Rail, req PostReq) (PostRes, error) {
  	var res miso.GnResp[PostRes]
  	err := miso.NewDynClient(rail, "/api/v1", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		var dat PostRes
  		return dat, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface PostReq {
    requestId?: string;
  }

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

  1() {
    let req: PostReq | null = null;
    this.http.post<any>(`/demo/api/v1`, req)
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

## POST /api/v2

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
  curl -X POST 'http://localhost:8080/api/v2' \
    -H 'Content-Type: application/json' \
    -d '{"requestId":""}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type PostReq struct {
  	RequestId string `json:"requestId"`
  }

  type PostRes struct {
  	ResultId string `json:"resultId"`
  	Time atom.Time `json:"time"`
  }

  func api2(rail miso.Rail, req *PostReq) (PostRes, error) {
  	var res miso.GnResp[PostRes]
  	err := miso.NewDynClient(rail, "/api/v2", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		var dat PostRes
  		return dat, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface PostReq {
    requestId?: string;
  }

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

  2() {
    let req: PostReq | null = null;
    this.http.post<any>(`/demo/api/v2`, req)
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

## POST /api/v3

- JSON Request:
    - "requestId": (string) 
- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (*api.PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v3' \
    -H 'Content-Type: application/json' \
    -d '{"requestId":""}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type PostReq struct {
  	RequestId string `json:"requestId"`
  }

  type PostRes struct {
  	ResultId string `json:"resultId"`
  	Time atom.Time `json:"time"`
  }

  func api3(rail miso.Rail, req *PostReq) (*PostRes, error) {
  	var res miso.GnResp[*PostRes]
  	err := miso.NewDynClient(rail, "/api/v3", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		return nil, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface PostReq {
    requestId?: string;
  }

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

  3() {
    let req: PostReq | null = null;
    this.http.post<any>(`/demo/api/v3`, req)
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

## POST /api/v4

- JSON Request:
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (*api.PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v4' \
    -H 'Content-Type: application/json' \
    -d '{"extras":[],"name":""}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type ApiReq struct {
  	Name string `json:"name"`
  	Extras []ApiReqExtra `json:"extras"`
  }

  type ApiReqExtra struct {
  	Special bool `json:"special"`
  }

  type PostRes struct {
  	ResultId string `json:"resultId"`
  	Time atom.Time `json:"time"`
  }

  func api4(rail miso.Rail, req ApiReq) (*PostRes, error) {
  	var res miso.GnResp[*PostRes]
  	err := miso.NewDynClient(rail, "/api/v4", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		return nil, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface ApiReq {
    name?: string;
    extras?: ApiReqExtra[];
  }

  export interface ApiReqExtra {
    special?: boolean;
  }

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

  4() {
    let req: ApiReq | null = null;
    this.http.post<any>(`/demo/api/v4`, req)
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

## POST /api/v5

- JSON Request:
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (*api.PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v5' \
    -H 'Content-Type: application/json' \
    -d '{"extras":[],"name":""}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type ApiReq struct {
  	Name string `json:"name"`
  	Extras []ApiReqExtra `json:"extras"`
  }

  type ApiReqExtra struct {
  	Special bool `json:"special"`
  }

  type PostRes struct {
  	ResultId string `json:"resultId"`
  	Time atom.Time `json:"time"`
  }

  func api5(rail miso.Rail, req *ApiReq) (*PostRes, error) {
  	var res miso.GnResp[*PostRes]
  	err := miso.NewDynClient(rail, "/api/v5", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		return nil, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface ApiReq {
    name?: string;
    extras?: ApiReqExtra[];
  }

  export interface ApiReqExtra {
    special?: boolean;
  }

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

  5() {
    let req: ApiReq | null = null;
    this.http.post<any>(`/demo/api/v5`, req)
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

## POST /api/v6

- JSON Request:
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (*api.PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v6' \
    -H 'Content-Type: application/json' \
    -d '{"extras":[],"name":""}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type ApiReq struct {
  	Name string `json:"name"`
  	Extras []ApiReqExtra `json:"extras"`
  }

  type ApiReqExtra struct {
  	Special bool `json:"special"`
  }

  type PostRes struct {
  	ResultId string `json:"resultId"`
  	Time atom.Time `json:"time"`
  }

  func api6(rail miso.Rail, req *ApiReq) (*PostRes, error) {
  	var res miso.GnResp[*PostRes]
  	err := miso.NewDynClient(rail, "/api/v6", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		return nil, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface ApiReq {
    name?: string;
    extras?: ApiReqExtra[];
  }

  export interface ApiReqExtra {
    special?: boolean;
  }

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

  6() {
    let req: ApiReq | null = null;
    this.http.post<any>(`/demo/api/v6`, req)
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

## POST /api/v7

- JSON Request:
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (ApiRes) response data
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v7' \
    -H 'Content-Type: application/json' \
    -d '{"extras":[],"name":""}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type ApiReq struct {
  	Name string `json:"name"`
  	Extras []ApiReqExtra `json:"extras"`
  }

  type ApiReqExtra struct {
  	Special bool `json:"special"`
  }

  type ApiRes struct {
  }

  func api7(rail miso.Rail, req *ApiReq) (ApiRes, error) {
  	var res miso.GnResp[ApiRes]
  	err := miso.NewDynClient(rail, "/api/v7", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		var dat ApiRes
  		return dat, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface ApiReq {
    name?: string;
    extras?: ApiReqExtra[];
  }

  export interface ApiReqExtra {
    special?: boolean;
  }

  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
    data?: ApiRes;                 // response data
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

  7() {
    let req: ApiReq | null = null;
    this.http.post<any>(`/demo/api/v7`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: ApiRes = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## POST /api/v8

- JSON Request:
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (*api.ApiRes) response data
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v8' \
    -H 'Content-Type: application/json' \
    -d '{"extras":[],"name":""}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type ApiReq struct {
  	Name string `json:"name"`
  	Extras []ApiReqExtra `json:"extras"`
  }

  type ApiReqExtra struct {
  	Special bool `json:"special"`
  }

  type ApiRes struct {
  }

  func api8(rail miso.Rail, req *ApiReq) (*ApiRes, error) {
  	var res miso.GnResp[*ApiRes]
  	err := miso.NewDynClient(rail, "/api/v8", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		return nil, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface ApiReq {
    name?: string;
    extras?: ApiReqExtra[];
  }

  export interface ApiReqExtra {
    special?: boolean;
  }

  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
    data?: ApiRes;                 // response data
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

  8() {
    let req: ApiReq | null = null;
    this.http.post<any>(`/demo/api/v8`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: ApiRes = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## POST /api/v9

- JSON Request:
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": ([]*api.ApiRes) response data
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v9' \
    -H 'Content-Type: application/json' \
    -d '{"extras":[],"name":""}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type ApiReq struct {
  	Name string `json:"name"`
  	Extras []ApiReqExtra `json:"extras"`
  }

  type ApiReqExtra struct {
  	Special bool `json:"special"`
  }

  type ApiRes struct {
  }

  func api9(rail miso.Rail, req *ApiReq) ([]*ApiRes, error) {
  	var res miso.GnResp[[]*ApiRes]
  	err := miso.NewDynClient(rail, "/api/v9", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		var dat []*ApiRes
  		return dat, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface ApiReq {
    name?: string;
    extras?: ApiReqExtra[];
  }

  export interface ApiReqExtra {
    special?: boolean;
  }

  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
    data?: ApiRes[];               // response data
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

  9() {
    let req: ApiReq | null = null;
    this.http.post<any>(`/demo/api/v9`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: ApiRes[] = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## POST /api/v10

- JSON Request:
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": ([]api.ApiRes) response data
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v10' \
    -H 'Content-Type: application/json' \
    -d '{"extras":[],"name":""}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type ApiReq struct {
  	Name string `json:"name"`
  	Extras []ApiReqExtra `json:"extras"`
  }

  type ApiReqExtra struct {
  	Special bool `json:"special"`
  }

  type ApiRes struct {
  }

  func api10(rail miso.Rail, req *ApiReq) ([]ApiRes, error) {
  	var res miso.GnResp[[]ApiRes]
  	err := miso.NewDynClient(rail, "/api/v10", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		var dat []ApiRes
  		return dat, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface ApiReq {
    name?: string;
    extras?: ApiReqExtra[];
  }

  export interface ApiReqExtra {
    special?: boolean;
  }

  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
    data?: ApiRes[];               // response data
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

  10() {
    let req: ApiReq | null = null;
    this.http.post<any>(`/demo/api/v10`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: ApiRes[] = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## POST /api/v11

- JSON Request:
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": ([]api.PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v11' \
    -H 'Content-Type: application/json' \
    -d '{"extras":[],"name":""}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type ApiReq struct {
  	Name string `json:"name"`
  	Extras []ApiReqExtra `json:"extras"`
  }

  type ApiReqExtra struct {
  	Special bool `json:"special"`
  }

  type PostRes struct {
  	ResultId string `json:"resultId"`
  	Time atom.Time `json:"time"`
  }

  func api11(rail miso.Rail, req *ApiReq) ([]PostRes, error) {
  	var res miso.GnResp[[]PostRes]
  	err := miso.NewDynClient(rail, "/api/v11", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		var dat []PostRes
  		return dat, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface ApiReq {
    name?: string;
    extras?: ApiReqExtra[];
  }

  export interface ApiReqExtra {
    special?: boolean;
  }

  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
    data?: PostRes[];
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

  11() {
    let req: ApiReq | null = null;
    this.http.post<any>(`/demo/api/v11`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: PostRes[] = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## POST /api/v12

- JSON Request: (array)
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": ([]api.PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v12' \
    -H 'Content-Type: application/json' \
    -d '[ {"extras":[],"name":""} ]'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type ApiReq struct {
  	Name string `json:"name"`
  	Extras []ApiReqExtra `json:"extras"`
  }

  type ApiReqExtra struct {
  	Special bool `json:"special"`
  }

  type PostRes struct {
  	ResultId string `json:"resultId"`
  	Time atom.Time `json:"time"`
  }

  func api12(rail miso.Rail, req []ApiReq) ([]PostRes, error) {
  	var res miso.GnResp[[]PostRes]
  	err := miso.NewDynClient(rail, "/api/v12", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		var dat []PostRes
  		return dat, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface ApiReq {
    name?: string;
    extras?: ApiReqExtra[];
  }

  export interface ApiReqExtra {
    special?: boolean;
  }

  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
    data?: PostRes[];
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

  12() {
    let req: ApiReq[] | null = null;
    this.http.post<any>(`/demo/api/v12`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: PostRes[] = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## POST /api/v13

- JSON Request: (array)
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v13' \
    -H 'Content-Type: application/json' \
    -d '[ {"extras":[],"name":""} ]'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type ApiReq struct {
  	Name string `json:"name"`
  	Extras []ApiReqExtra `json:"extras"`
  }

  type ApiReqExtra struct {
  	Special bool `json:"special"`
  }

  func api13(rail miso.Rail, req []ApiReq) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynClient(rail, "/api/v13", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		return err
  	}
  	return nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface ApiReq {
    name?: string;
    extras?: ApiReqExtra[];
  }

  export interface ApiReqExtra {
    special?: boolean;
  }

  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
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

  13() {
    let req: ApiReq[] | null = null;
    this.http.post<any>(`/demo/api/v13`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## POST /api/v14

- JSON Request:
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": ([]api.PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v14' \
    -H 'Content-Type: application/json' \
    -d '{"extras":[],"name":""}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type ApiReq struct {
  	Name string `json:"name"`
  	Extras []ApiReqExtra `json:"extras"`
  }

  type ApiReqExtra struct {
  	Special bool `json:"special"`
  }

  type PostRes struct {
  	ResultId string `json:"resultId"`
  	Time atom.Time `json:"time"`
  }

  func api14(rail miso.Rail, req ApiReq) ([]PostRes, error) {
  	var res miso.GnResp[[]PostRes]
  	err := miso.NewDynClient(rail, "/api/v14", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		var dat []PostRes
  		return dat, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface ApiReq {
    name?: string;
    extras?: ApiReqExtra[];
  }

  export interface ApiReqExtra {
    special?: boolean;
  }

  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
    data?: PostRes[];
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

  14() {
    let req: ApiReq | null = null;
    this.http.post<any>(`/demo/api/v14`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: PostRes[] = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## GET /api/v15

- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": ([]api.PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
- cURL:
  ```sh
  curl -X GET 'http://localhost:8080/api/v15'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type PostRes struct {
  	ResultId string `json:"resultId"`
  	Time atom.Time `json:"time"`
  }

  func api15(rail miso.Rail) ([]PostRes, error) {
  	var res miso.GnResp[[]PostRes]
  	err := miso.NewDynClient(rail, "/api/v15", "demo").
  		Get().
  		Json(&res)
  	if err != nil {
  		var dat []PostRes
  		return dat, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
    data?: PostRes[];
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

  15() {
    this.http.get<any>(`/demo/api/v15`)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: PostRes[] = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## GET /api/v16

- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (PageRes[github.com/curtisnewbie/miso/demo/api.PostRes]) response data
      - "paging": (Paging) pagination parameters
        - "limit": (int) page limit
        - "page": (int) page number, 1-based
        - "total": (int) total count
      - "payload": ([]api.PostRes) payload values in current page
        - "resultId": (string) 
        - "time": (int64) 
- cURL:
  ```sh
  curl -X GET 'http://localhost:8080/api/v16'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go

  type PostRes struct {
  	ResultId string `json:"resultId"`
  	Time atom.Time `json:"time"`
  }

  func api16(rail miso.Rail) (miso.PageRes[PostRes], error) {
  	var res miso.GnResp[miso.PageRes[PostRes]]
  	err := miso.NewDynClient(rail, "/api/v16", "demo").
  		Get().
  		Json(&res)
  	if err != nil {
  		var dat miso.PageRes[PostRes]
  		return dat, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
    data?: PageRes;
  }

  export interface PageRes {
    paging?: Paging;
    payload?: PostRes[];
  }

  export interface Paging {
    limit?: number;                // page limit
    page?: number;                 // page number, 1-based
    total?: number;                // total count
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

  16() {
    this.http.get<any>(`/demo/api/v16`)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: PageRes = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

- Angular NgTable Demo:
  ```html
  <table mat-table [dataSource]="tabdata" class="mb-4" style="width: 100%;">
  	<ng-container matColumnDef="resultId">
  		<th mat-header-cell *matHeaderCellDef> ResultId </th>
  		<td mat-cell *matCellDef="let u"> {{u.resultId}} </td>
  	</ng-container>
  	<ng-container matColumnDef="time">
  		<th mat-header-cell *matHeaderCellDef> Time </th>
  		<td mat-cell *matCellDef="let u"> {{u.time | date: 'yyyy-MM-dd HH:mm:ss'}} </td>
  	</ng-container>
  	<tr mat-row *matRowDef="let row; columns: ['resultId','time'];"></tr>
  	<tr mat-header-row *matHeaderRowDef="['resultId','time']"></tr>
  </table>
  ```

## GET /api/v17

- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": ([]api.PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
- cURL:
  ```sh
  curl -X GET 'http://localhost:8080/api/v17'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type PostRes struct {
  	ResultId string `json:"resultId"`
  	Time atom.Time `json:"time"`
  }

  func api17(rail miso.Rail) ([]PostRes, error) {
  	var res miso.GnResp[[]PostRes]
  	err := miso.NewDynClient(rail, "/api/v17", "demo").
  		Get().
  		Json(&res)
  	if err != nil {
  		var dat []PostRes
  		return dat, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
    data?: PostRes[];
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

  17() {
    this.http.get<any>(`/demo/api/v17`)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: PostRes[] = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## POST /api/v18

- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v18'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  func api18(rail miso.Rail) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynClient(rail, "/api/v18", "demo").
  		Post(nil).
  		Json(&res)
  	if err != nil {
  		return err
  	}
  	return nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
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

  18() {
    this.http.post<any>(`/demo/api/v18`, null)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## GET /api/v19

- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
- cURL:
  ```sh
  curl -X GET 'http://localhost:8080/api/v19'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  func api19(rail miso.Rail) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynClient(rail, "/api/v19", "demo").
  		Get().
  		Json(&res)
  	if err != nil {
  		return err
  	}
  	return nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
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

  19() {
    this.http.get<any>(`/demo/api/v19`)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## POST /api/v20

- JSON Request:
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v20' \
    -H 'Content-Type: application/json' \
    -d '{"extras":[],"name":""}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type ApiReq struct {
  	Name string `json:"name"`
  	Extras []ApiReqExtra `json:"extras"`
  }

  type ApiReqExtra struct {
  	Special bool `json:"special"`
  }

  func api20(rail miso.Rail, req ApiReq) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynClient(rail, "/api/v20", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		return err
  	}
  	return nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface ApiReq {
    name?: string;
    extras?: ApiReqExtra[];
  }

  export interface ApiReqExtra {
    special?: boolean;
  }

  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
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

  20() {
    let req: ApiReq | null = null;
    this.http.post<any>(`/demo/api/v20`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## POST /api/v21

- JSON Request:
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v21' \
    -H 'Content-Type: application/json' \
    -d '{"extras":[],"name":""}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type ApiReq struct {
  	Name string `json:"name"`
  	Extras []ApiReqExtra `json:"extras"`
  }

  type ApiReqExtra struct {
  	Special bool `json:"special"`
  }

  func api21(rail miso.Rail, req ApiReq) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynClient(rail, "/api/v21", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		return err
  	}
  	return nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface ApiReq {
    name?: string;
    extras?: ApiReqExtra[];
  }

  export interface ApiReqExtra {
    special?: boolean;
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

  21() {
    let req: ApiReq | null = null;
    this.http.post<any>(`/demo/api/v21`, req)
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

## POST /api/v22

- JSON Request:
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
- JSON Response:
    - "resultId": (string) 
    - "time": (int64) 
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v22' \
    -H 'Content-Type: application/json' \
    -d '{"extras":[],"name":""}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type ApiReq struct {
  	Name string `json:"name"`
  	Extras []ApiReqExtra `json:"extras"`
  }

  type ApiReqExtra struct {
  	Special bool `json:"special"`
  }

  type PostRes struct {
  	ResultId string `json:"resultId"`
  	Time atom.Time `json:"time"`
  }

  func api22(rail miso.Rail, req ApiReq) (PostRes, error) {
  	var res miso.GnResp[PostRes]
  	err := miso.NewDynClient(rail, "/api/v22", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		var dat PostRes
  		return dat, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface ApiReq {
    name?: string;
    extras?: ApiReqExtra[];
  }

  export interface ApiReqExtra {
    special?: boolean;
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

  22() {
    let req: ApiReq | null = null;
    this.http.post<PostRes>(`/demo/api/v22`, req)
      .subscribe({
        next: (resp) => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## POST /api/v23

- JSON Response:
    - "resultId": (string) 
    - "time": (int64) 
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v23'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type PostRes struct {
  	ResultId string `json:"resultId"`
  	Time atom.Time `json:"time"`
  }

  func api23(rail miso.Rail) (PostRes, error) {
  	var res miso.GnResp[PostRes]
  	err := miso.NewDynClient(rail, "/api/v23", "demo").
  		Post(nil).
  		Json(&res)
  	if err != nil {
  		var dat PostRes
  		return dat, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
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

  23() {
    this.http.post<PostRes>(`/demo/api/v23`, null)
      .subscribe({
        next: (resp) => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## POST /api/v24

- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v24'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  func api24(rail miso.Rail) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynClient(rail, "/api/v24", "demo").
  		Post(nil).
  		Json(&res)
  	if err != nil {
  		return err
  	}
  	return nil
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

  24() {
    this.http.post<any>(`/demo/api/v24`, null)
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

## POST /api/v25

- JSON Response:
    - "resultId": (string) 
    - "time": (int64) 
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v25'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type PostRes struct {
  	ResultId string `json:"resultId"`
  	Time atom.Time `json:"time"`
  }

  func api25(rail miso.Rail) (PostRes, error) {
  	var res miso.GnResp[PostRes]
  	err := miso.NewDynClient(rail, "/api/v25", "demo").
  		Post(nil).
  		Json(&res)
  	if err != nil {
  		var dat PostRes
  		return dat, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
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

  25() {
    this.http.post<PostRes>(`/demo/api/v25`, null)
      .subscribe({
        next: (resp) => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## OPTIONS /api/v26

- cURL:
  ```sh
  curl -X OPTIONS 'http://localhost:8080/api/v26'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  func api26(rail miso.Rail) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynClient(rail, "/api/v26", "demo").
  		Options().
  		Json(&res)
  	if err != nil {
  		return err
  	}
  	return nil
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

  26() {
    this.http.options<any>(`/demo/api/v26`)
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

## HEAD /api/v27

- cURL:
  ```sh
  curl -X HEAD 'http://localhost:8080/api/v27'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  func api27(rail miso.Rail) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynClient(rail, "/api/v27", "demo").
  		Head().
  		Json(&res)
  	if err != nil {
  		return err
  	}
  	return nil
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

  27() {
    this.http.head<any>(`/demo/api/v27`)
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

## PATCH /api/v28

- cURL:
  ```sh
  curl -X PATCH 'http://localhost:8080/api/v28'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  func api28(rail miso.Rail) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynClient(rail, "/api/v28", "demo").
  		Patch().
  		Json(&res)
  	if err != nil {
  		return err
  	}
  	return nil
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

  28() {
    this.http.patch<any>(`/demo/api/v28`)
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

## CONNECT /api/v29

- cURL:
  ```sh
  curl -X CONNECT 'http://localhost:8080/api/v29'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  func api29(rail miso.Rail) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynClient(rail, "/api/v29", "demo").
  		Connect().
  		Json(&res)
  	if err != nil {
  		return err
  	}
  	return nil
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

  29() {
    this.http.connect<any>(`/demo/api/v29`)
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

## TRACE /api/v30

- cURL:
  ```sh
  curl -X TRACE 'http://localhost:8080/api/v30'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  func api30(rail miso.Rail) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynClient(rail, "/api/v30", "demo").
  		Trace().
  		Json(&res)
  	if err != nil {
  		return err
  	}
  	return nil
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

  30() {
    this.http.trace<any>(`/demo/api/v30`)
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

## POST /api/v31

- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v31'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type EmptyReq struct {
  }

  func api31(rail miso.Rail, req EmptyReq) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynClient(rail, "/api/v31", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		return err
  	}
  	return nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface EmptyReq {
  }

  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
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

  31() {
    let req: EmptyReq | null = null;
    this.http.post<any>(`/demo/api/v31`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## POST /api/v32

- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (map[string]int32) response data
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v32'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type EmptyReq struct {
  }

  func api32(rail miso.Rail, req EmptyReq) (map[string]int32, error) {
  	var res miso.GnResp[map[string]int32]
  	err := miso.NewDynClient(rail, "/api/v32", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		var dat map[string]int32
  		return dat, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface EmptyReq {
  }

  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
    data?: Map<string,number>;     // response data
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

  32() {
    let req: EmptyReq | null = null;
    this.http.post<any>(`/demo/api/v32`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: Map<string,number> = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## POST /api/v33

- JSON Request:
    - "time": (int64) 
    - "amt": (string) 
    - "set": ([]string) 
- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (*api.PostRes2) response data
      - "time": (int64) 
      - "amt": (string) 
      - "amtPtr": (*string) 
      - "set": ([]string) 
      - "setPtr": ([]string) 
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/api/v33' \
    -H 'Content-Type: application/json' \
    -d '{"amt":"","set":null,"time":0}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type ApiReq2 struct {
  	Time atom.Time `json:"time"`
  	Amt money.Amt `json:"amt"`
  	Set hash.Set[string] `json:"set"`
  }

  type PostRes2 struct {
  	Time atom.Time `json:"time"`
  	Amt money.Amt `json:"amt"`
  	AmtPtr *money.Amt `json:"amtPtr"`
  	Set hash.Set[string] `json:"set"`
  	SetPtr *hash.Set[string] `json:"setPtr"`
  }

  func api33(rail miso.Rail, req *ApiReq2) (*PostRes2, error) {
  	var res miso.GnResp[*PostRes2]
  	err := miso.NewDynClient(rail, "/api/v33", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		return nil, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface ApiReq2 {
    time?: number;
    amt?: string;
    set?: string[];
  }

  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
    data?: PostRes2;
  }

  export interface PostRes2 {
    time?: number;
    amt?: string;
    amtPtr?: string;
    set?: string[];
    setPtr?: string[];
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

  33() {
    let req: ApiReq2 | null = null;
    this.http.post<any>(`/demo/api/v33`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: PostRes2 = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

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
  	RequestId string `json:"requestId"`
  }

  type PostRes struct {
  	ResultId string `json:"resultId"`
  	Time atom.Time `json:"time"`
  }

  // Post demo stuff
  func SendPostReq(rail miso.Rail, req PostReq, authorization string) (PostRes, error) {
  	var res miso.GnResp[PostRes]
  	err := miso.NewDynClient(rail, "/open/api/demo/grouped/open/api/demo/post", "demo").
  		AddHeader("authorization", authorization).
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		var dat PostRes
  		return dat, err
  	}
  	return res.Data, nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface PostReq {
    requestId?: string;
  }

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

## GET /debug/trace/recorder/run

- Description: Start FlightRecorder. Recorded result is written to trace.out when it's finished or stopped.
- Query Parameter:
  - "duration": Duration of the flight recording. Required. Duration cannot exceed 30 min.
- cURL:
  ```sh
  curl -X GET 'http://localhost:8080/debug/trace/recorder/run?duration='
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  // Start FlightRecorder. Recorded result is written to trace.out when it's finished or stopped.
  func SendRequest(rail miso.Rail, duration string) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynClient(rail, "/debug/trace/recorder/run", "demo").
  		AddQuery("duration", duration).
  		Get().
  		Json(&res)
  	if err != nil {
  		return err
  	}
  	return nil
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
    let duration: any | null = null;
    this.http.get<any>(`/demo/debug/trace/recorder/run?duration=${duration}`)
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

## GET /debug/trace/recorder/snapshot

- Description: FlightRecorder take snapshot. Recorded result is written to trace.out.
- cURL:
  ```sh
  curl -X GET 'http://localhost:8080/debug/trace/recorder/snapshot'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  // FlightRecorder take snapshot. Recorded result is written to trace.out.
  func SendRequest(rail miso.Rail) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynClient(rail, "/debug/trace/recorder/snapshot", "demo").
  		Get().
  		Json(&res)
  	if err != nil {
  		return err
  	}
  	return nil
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
    this.http.get<any>(`/demo/debug/trace/recorder/snapshot`)
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

## GET /debug/trace/recorder/stop

- Description: Stop existing FlightRecorder session.
- cURL:
  ```sh
  curl -X GET 'http://localhost:8080/debug/trace/recorder/stop'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  // Stop existing FlightRecorder session.
  func SendRequest(rail miso.Rail) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynClient(rail, "/debug/trace/recorder/stop", "demo").
  		Get().
  		Json(&res)
  	if err != nil {
  		return err
  	}
  	return nil
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
    this.http.get<any>(`/demo/debug/trace/recorder/stop`)
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

## POST /debug/task/disable-workers

- Description: Manually Disable Distributed Task Worker By Name. Use '*' as a special placeholder for all tasks currently registered. For debugging only.
- JSON Request:
    - "tasks": ([]string) 
- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/debug/task/disable-workers' \
    -H 'Content-Type: application/json' \
    -d '{"tasks":[]}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type disableTaskWorkerReq struct {
  	Tasks []string `json:"tasks"`
  }

  // Manually Disable Distributed Task Worker By Name. Use '*' as a special placeholder for all tasks currently registered. For debugging only.
  func SendDisableTaskWorkerReq(rail miso.Rail, req disableTaskWorkerReq) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynClient(rail, "/debug/task/disable-workers", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		return err
  	}
  	return nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface disableTaskWorkerReq {
    tasks?: string[];
  }

  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
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

  sendDisableTaskWorkerReq() {
    let req: disableTaskWorkerReq | null = null;
    this.http.post<any>(`/demo/debug/task/disable-workers`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
  }
  ```

## POST /debug/task/enable-workers

- Description: Manually enable previously disabled Distributed Task Worker By Name. Use '*' as a special placeholder for all tasks currently registered. For debugging only.
- JSON Request:
    - "tasks": ([]string) 
- JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
- cURL:
  ```sh
  curl -X POST 'http://localhost:8080/debug/task/enable-workers' \
    -H 'Content-Type: application/json' \
    -d '{"tasks":[]}'
  ```

- Miso HTTP Client (experimental, demo may not work):
  ```go
  type disableTaskWorkerReq struct {
  	Tasks []string `json:"tasks"`
  }

  // Manually enable previously disabled Distributed Task Worker By Name. Use '*' as a special placeholder for all tasks currently registered. For debugging only.
  func SendDisableTaskWorkerReq(rail miso.Rail, req disableTaskWorkerReq) error {
  	var res miso.GnResp[any]
  	err := miso.NewDynClient(rail, "/debug/task/enable-workers", "demo").
  		PostJson(req).
  		Json(&res)
  	if err != nil {
  		return err
  	}
  	return nil
  }
  ```

- JSON Request / Response Object In TypeScript:
  ```ts
  export interface disableTaskWorkerReq {
    tasks?: string[];
  }

  export interface Resp {
    errorCode?: string;            // error code
    msg?: string;                  // message
    error?: boolean;               // whether the request was successful
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

  sendDisableTaskWorkerReq() {
    let req: disableTaskWorkerReq | null = null;
    this.http.post<any>(`/demo/debug/task/enable-workers`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
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
    - "Value": (string) 
