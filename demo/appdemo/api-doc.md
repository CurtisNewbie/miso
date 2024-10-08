# API Endpoints

- POST /api/v1
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

  - JSON Request Object In TypeScript:
    ```ts
    export interface PostReq {
      requestId?: string
    }
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: PostRes
    }

    export interface PostRes {
      resultId?: string
      time?: number
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
    ```

- POST /api/v2
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

  - JSON Request Object In TypeScript:
    ```ts
    export interface PostReq {
      requestId?: string
    }
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: PostRes
    }

    export interface PostRes {
      resultId?: string
      time?: number
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
    ```

- POST /api/v3
  - JSON Request:
    - "requestId": (string) 
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (*main.PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
  - cURL:
    ```sh
    curl -X POST 'http://localhost:8080/api/v3' \
      -H 'Content-Type: application/json' \
      -d '{"requestId":""}'
    ```

  - JSON Request Object In TypeScript:
    ```ts
    export interface PostReq {
      requestId?: string
    }
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: PostRes
    }

    export interface PostRes {
      resultId?: string
      time?: number
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
    ```

- POST /api/v4
  - JSON Request:
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (*main.PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
  - cURL:
    ```sh
    curl -X POST 'http://localhost:8080/api/v4' \
      -H 'Content-Type: application/json' \
      -d '{"extras":{"special":false},"name":""}'
    ```

  - JSON Request Object In TypeScript:
    ```ts
    export interface ApiReq {
      name?: string
      extras?: ApiReqExtra[]
    }

    export interface ApiReqExtra {
      special?: boolean
    }
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: PostRes
    }

    export interface PostRes {
      resultId?: string
      time?: number
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
    ```

- POST /api/v5
  - JSON Request:
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (*main.PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
  - cURL:
    ```sh
    curl -X POST 'http://localhost:8080/api/v5' \
      -H 'Content-Type: application/json' \
      -d '{"extras":{"special":false},"name":""}'
    ```

  - JSON Request Object In TypeScript:
    ```ts
    export interface ApiReq {
      name?: string
      extras?: ApiReqExtra[]
    }

    export interface ApiReqExtra {
      special?: boolean
    }
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: PostRes
    }

    export interface PostRes {
      resultId?: string
      time?: number
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
    ```

- POST /api/v6
  - JSON Request:
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (*main.PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
  - cURL:
    ```sh
    curl -X POST 'http://localhost:8080/api/v6' \
      -H 'Content-Type: application/json' \
      -d '{"extras":{"special":false},"name":""}'
    ```

  - JSON Request Object In TypeScript:
    ```ts
    export interface ApiReq {
      name?: string
      extras?: ApiReqExtra[]
    }

    export interface ApiReqExtra {
      special?: boolean
    }
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: PostRes
    }

    export interface PostRes {
      resultId?: string
      time?: number
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
    ```

- POST /api/v7
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
      -d '{"extras":{"special":false},"name":""}'
    ```

  - JSON Request Object In TypeScript:
    ```ts
    export interface ApiReq {
      name?: string
      extras?: ApiReqExtra[]
    }

    export interface ApiReqExtra {
      special?: boolean
    }
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: ApiRes                  // response data
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
    ```

- POST /api/v8
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
      -d '{"extras":{"special":false},"name":""}'
    ```

  - JSON Request Object In TypeScript:
    ```ts
    export interface ApiReq {
      name?: string
      extras?: ApiReqExtra[]
    }

    export interface ApiReqExtra {
      special?: boolean
    }
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: *api.ApiRes             // response data
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

    let req: ApiReq | null = null;
    this.http.post<any>(`/demo/api/v8`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: *api.ApiRes = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- POST /api/v9
  - JSON Request:
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (*[]api.ApiRes) response data
  - cURL:
    ```sh
    curl -X POST 'http://localhost:8080/api/v9' \
      -H 'Content-Type: application/json' \
      -d '{"extras":{"special":false},"name":""}'
    ```

  - JSON Request Object In TypeScript:
    ```ts
    export interface ApiReq {
      name?: string
      extras?: ApiReqExtra[]
    }

    export interface ApiReqExtra {
      special?: boolean
    }
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: *[]api.ApiRes           // response data
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

    let req: ApiReq | null = null;
    this.http.post<any>(`/demo/api/v9`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: *[]api.ApiRes = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- POST /api/v10
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
      -d '{"extras":{"special":false},"name":""}'
    ```

  - JSON Request Object In TypeScript:
    ```ts
    export interface ApiReq {
      name?: string
      extras?: ApiReqExtra[]
    }

    export interface ApiReqExtra {
      special?: boolean
    }
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: api.ApiRes[]            // response data
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

    let req: ApiReq | null = null;
    this.http.post<any>(`/demo/api/v10`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: api.ApiRes[] = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- POST /api/v11
  - JSON Request:
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": ([]main.PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
  - cURL:
    ```sh
    curl -X POST 'http://localhost:8080/api/v11' \
      -H 'Content-Type: application/json' \
      -d '{"extras":{"special":false},"name":""}'
    ```

  - JSON Request Object In TypeScript:
    ```ts
    export interface ApiReq {
      name?: string
      extras?: ApiReqExtra[]
    }

    export interface ApiReqExtra {
      special?: boolean
    }
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: PostRes[]
    }

    export interface PostRes {
      resultId?: string
      time?: number
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
    ```

- POST /api/v12
  - JSON Request: (array)
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": ([]main.PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
  - cURL:
    ```sh
    curl -X POST 'http://localhost:8080/api/v12' \
      -H 'Content-Type: application/json' \
      -d '[ {"extras":{"special":false},"name":""} ]'
    ```

  - JSON Request Object In TypeScript:
    ```ts
    export interface ApiReq {
      name?: string
      extras?: ApiReqExtra[]
    }

    export interface ApiReqExtra {
      special?: boolean
    }
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: PostRes[]
    }

    export interface PostRes {
      resultId?: string
      time?: number
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
    ```

- POST /api/v13
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
      -d '[ {"extras":{"special":false},"name":""} ]'
    ```

  - JSON Request Object In TypeScript:
    ```ts
    export interface ApiReq {
      name?: string
      extras?: ApiReqExtra[]
    }

    export interface ApiReqExtra {
      special?: boolean
    }
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
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
    ```

- POST /api/v14
  - JSON Request:
    - "name": (string) 
    - "extras": ([]api.ApiReqExtra) 
      - "special": (bool) 
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": ([]main.PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
  - cURL:
    ```sh
    curl -X POST 'http://localhost:8080/api/v14' \
      -H 'Content-Type: application/json' \
      -d '{"extras":{"special":false},"name":""}'
    ```

  - JSON Request Object In TypeScript:
    ```ts
    export interface ApiReq {
      name?: string
      extras?: ApiReqExtra[]
    }

    export interface ApiReqExtra {
      special?: boolean
    }
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: PostRes[]
    }

    export interface PostRes {
      resultId?: string
      time?: number
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
    ```

- GET /api/v15
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": ([]main.PostRes) response data
      - "resultId": (string) 
      - "time": (int64) 
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8080/api/v15'
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: PostRes[]
    }

    export interface PostRes {
      resultId?: string
      time?: number
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
    ```

- GET /api/v16
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (PageRes[main.PostRes]) response data
      - "paging": (Paging) pagination parameters
        - "limit": (int) page limit
        - "page": (int) page number, 1-based
        - "total": (int) total count
      - "payload": ([]main.PostRes) payload values in current page
        - "resultId": (string) 
        - "time": (int64) 
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8080/api/v16'
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: PageRes
    }

    export interface PageRes {
      paging?: Paging
      payload?: PostRes[]
    }

    export interface Paging {
      limit?: number                 // page limit
      page?: number                  // page number, 1-based
      total?: number                 // total count
    }

    export interface PostRes {
      resultId?: string
      time?: number
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
    ```

- POST /open/api/demo/grouped/open/api/demo/post
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

  - JSON Request Object In TypeScript:
    ```ts
    export interface PostReq {
      requestId?: string
    }
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: PostRes
    }

    export interface PostRes {
      resultId?: string
      time?: number
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
    ```

- POST /open/api/demo/grouped/subgroup/post1
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

  - JSON Request Object In TypeScript:
    ```ts
    export interface PostReq {
      requestId?: string
    }
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: PostRes
    }

    export interface PostRes {
      resultId?: string
      time?: number
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
    ```

- GET /metrics
  - Description: Collect prometheus metrics information
  - Header Parameter:
    - "Authorization": Basic authorization if enabled
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8080/metrics' \
      -H 'Authorization: '
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

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
    ```

- GET /debug/pprof
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8080/debug/pprof'
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    this.http.get<any>(`/demo/debug/pprof`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /debug/pprof/:name
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8080/debug/pprof/:name'
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    this.http.get<any>(`/demo/debug/pprof/:name`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /debug/pprof/cmdline
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8080/debug/pprof/cmdline'
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    this.http.get<any>(`/demo/debug/pprof/cmdline`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /debug/pprof/profile
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8080/debug/pprof/profile'
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    this.http.get<any>(`/demo/debug/pprof/profile`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /debug/pprof/symbol
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8080/debug/pprof/symbol'
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    this.http.get<any>(`/demo/debug/pprof/symbol`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /debug/pprof/trace
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8080/debug/pprof/trace'
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    this.http.get<any>(`/demo/debug/pprof/trace`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /doc/api
  - Description: Serve the generated API documentation webpage
  - Expected Access Scope: PUBLIC
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8080/doc/api'
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    this.http.get<any>(`/demo/doc/api`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

# Event Pipelines

- DemoPipeline
  - Description: This is a demo pipeline
  - RabbitMQ Queue: `demo:pipeline`
  - RabbitMQ Exchange: `demo:pipeline`
  - RabbitMQ RoutingKey: `#`
  - Event Payload: (array)
    - "value": (string) 
