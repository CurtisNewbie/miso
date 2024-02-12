# API Documentation Generation

In non-prod mode (`mode.production: false`), an API documentation is automatically generated and exposed on `/doc/api` endpoint. If `server.port` is 8080, then the generated documentation is accessible through: `http://localhost:8080/doc/api`.

There are two types of documentation generated, one is simply a webpage rendered by browser, another one is the markdown version that can be copied and saved to some README.md files.

Without providing extra information, the generated documentation may not be very helpful. One may describe the endpoint as follows to provide extra metadata:

```go
miso.RawGet("/file/raw", TempKeyDownloadFileEp).
    Desc(`
        File download using temporary file key. This endpoint is expected to be accessible publicly without 
        authorization, since a temporary file_key is generated and used.
    `).
    Public().
    DocQueryParam("key", "temporary file key").
    Build()

miso.Put("/file", UploadFileEp).
    Desc("Fstore file upload. A temporary file_id is returned, which should be used to exchange the real file_id").
    Resource(ResCodeFstoreUpload).
    DocHeader("filename", "name of the uploaded file").
    DocJsonResp(miso.GnResp[string]{}).
    Build()

miso.IGet("/file/info", GetFileInfoEp).
    Desc("Fetch file info").
    DocQueryParam("uploadFileId", "temporary file_id returned for the newly uploaded file").
    DocQueryParam("fileId", "actual file_id of the file record").
    DocJsonResp(miso.GnResp[api.FstoreFile]{}).
    Build()
```