# API Documentation Generation

In non-prod mode (`mode.production: false`), an API documentation is automatically generated and exposed on `/doc/api` endpoint. If `server.port` is 8080, then the generated documentation is accessible through: `http://localhost:8080/doc/api`.

There are two types of documentation generated, one is simply a static webpage rendered, another one is the markdown version that can be copied and pasted to some README.md files.

miso tries its best to guess all the required parameters from your endpoints, it generates doc about your request/reponse in JSON format, including query parameters, headers and so on. As it continually improves, miso now can even create a demo curl script, as well as typescript object definitions for you. But of course, you may provide extra information to describe the endpoints:

e.g.,

```go
miso.RawGet("/file/raw", TempKeyDownloadFileEp).
    Desc(`
        File download using temporary file key. This endpoint is expected to be accessible publicly without
        authorization, since a temporary file_key is generated and used.
    `).
    Public().
    DocQueryParam("key", "temporary file key")

miso.Put("/file", UploadFileEp).
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
	UplTime    miso.ETime  `json:"uplTime" desc:"upload time"`
	LogDelTime *miso.ETime `json:"logDelTime" desc:"logically deleted at"`
	PhyDelTime *miso.ETime `json:"phyDelTime" desc:"physically deleted at"`
}

// endpoint
miso.IGet("/file/info", GetFileInfoEp).
    Desc("Fetch file info")

// handler
func GetFileInfoEp(inb *miso.Inbound, req FileInfoReq) (api.FstoreFile, error) {
    // ...
}
```

The resulting documentation looks like the following (this is the markdown version, but the web page version is almost the same):

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

---