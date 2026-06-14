# API Documentation Generation

API documentation is generated statically using the `misoapi` CLI tool. Run it in your project directory:

```bash
go install github.com/curtisnewbie/miso/cmd/misoapi@v0.4.13
misoapi
```

This scans your codebase for handler functions annotated with `misoapi-*` comments, generates endpoint registration code in `misoapi_generated.go`, and produces API documentation (markdown format). No runtime `/doc/api` endpoint is needed.

misoapi infers request/response types from your handler signatures, including query parameters, headers, JSON bodies, etc. It generates curl examples, TypeScript type definitions, and Angular HttpClient demos. You can provide additional metadata via endpoint configuration:

e.g.,

```go
miso.HttpGet("/file/raw", miso.RawHandler(TempKeyDownloadFileEp)).
    Desc(`
        File download using temporary file key. This endpoint is expected to be accessible publicly without
        authorization, since a temporary file_key is generated and used.
    `).
    Public().
    DocQueryParam("key", "temporary file key")

miso.HttpPut("/file", miso.ResHandler(UploadFileEp)).
    Desc("Fstore file upload. A temporary file_id is returned, which should be used to exchange the real file_id").
    Resource(ResCodeFstoreUpload).
    DocHeader("filename", "name of the uploaded file")
```

With generics, the generated api doc is actually quite good, the following is a demonstration:

```go
// the request object
type FileInfoReq struct {
	FileId       string `form:"fileId" desc:"actual file_id of the file record"`
	UploadFileId string `form:"uploadFileId" desc:"temporary file_id returned when uploading files"`
}

// the response obejct
type FstoreFile struct {
	FileId     string      `json:"fileId" desc:"file unique identifier"`
	Name       string      `json:"name" desc:"file name"`
	Status     string      `json:"status" desc:"status, 'NORMAL', 'LOG_DEL' (logically deleted), 'PHY_DEL' (physically deleted)"`
	Size       int64       `json:"size" desc:"file size in bytes"`
	Md5        string      `json:"md5" desc:"MD5 checksum"`
	UplTime    util.ETime  `json:"uplTime" desc:"upload time"`
	LogDelTime *util.ETime `json:"logDelTime" desc:"logically deleted at"`
	PhyDelTime *util.ETime `json:"phyDelTime" desc:"physically deleted at"`
}

// endpoint
miso.HttpGet("/file/info", miso.AutoHandler(GetFileInfoEp)).
    Desc("Fetch file info")

// handler
func GetFileInfoEp(inb *miso.Inbound, req FileInfoReq) (api.FstoreFile, error) {
    // ...
}
```

The resulting markdown documentation produced by misoapi looks like:

- GET /file/info
  - Description: Fetch file info
  - Query Parameter:
    - "fileId": actual file_id of the file record
    - "uploadFileId": temporary file_id returned when uploading files
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (FstoreFile) response data
      - "fileId": (string) file unique identifier
      - "name": (string) file name
      - "status": (string) status, 'NORMAL', 'LOG_DEL' (logically deleted), 'PHY_DEL' (physically deleted)
      - "size": (int64) file size in bytes
      - "md5": (string) MD5 checksum
      - "uplTime": (int64) upload time
      - "logDelTime": (int64) logically deleted at
      - "phyDelTime": (int64) physically deleted at
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8084/file/info?fileId=&uploadFileId='
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: FstoreFile
    }
    export interface FstoreFile {
      fileId?: string                // file unique identifier
      name?: string                  // file name
      status?: string                // status, 'NORMAL', 'LOG_DEL' (logically deleted), 'PHY_DEL' (physically deleted)
      size?: number                  // file size in bytes
      md5?: string                   // MD5 checksum
      uplTime?: number               // upload time
      logDelTime?: number            // logically deleted at
      phyDelTime?: number            // physically deleted at
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

    let fileId: any | null = null;
    let uploadFileId: any | null = null;
    this.http.get<any>(`/file/info?fileId=${fileId}&uploadFileId=${uploadFileId}`)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: FstoreFile = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```
