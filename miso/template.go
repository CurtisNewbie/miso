package miso

import (
	"embed"
	"html/template"
	"io"
	"net/http"
	"path"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	tmplMapOnce sync.Once
	tmplMap     map[string]*template.Template
	tmplMapMu   sync.RWMutex
)

func ServeTempl(inb *Inbound, fs embed.FS, tmplName string, data any) {
	w, _ := inb.Unwrap()
	MustCompile(fs, tmplName).Execute(w, data)
}

func ServeStatic(inb *Inbound, fs embed.FS, file string) {
	w, _ := inb.Unwrap()
	f, err := fs.Open(file)
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(w, f); err != nil {
		panic(err)
	}
}

// Prepare to serve static files in embedded fs.
//
// Static files are served by paths that prefixes '/static'.
//
// Notice that index.html must be renamed to index.htm or else it won't work.
func PrepareWebStaticFs(fs embed.FS, dir string) {
	serveStaticFile := func(c *gin.Context, fp string) {
		c.FileFromFS(path.Join(dir, fp), http.FS(fs))
	}

	setNoRouteHandler(func(ctx *gin.Context, rail Rail) {
		// why are we using index.htm instead of index.html.
		//
		// https://stackoverflow.com/questions/69462376/serving-react-static-files-in-golang-gin-gonic-using-goembed-giving-404-error-o
		// https://cs.opensource.google/go/go/+/refs/tags/go1.21.5:src/net/http/fs.go;l=604
		// https://github.com/gin-contrib/static/issues/19
		if ctx.Request.Method == "GET" {
			if ctx.Request.RequestURI == "/" || ctx.Request.RequestURI == "/static/" || ctx.Request.RequestURI == "/static/index.html" {
				ctx.Redirect(http.StatusTemporaryRedirect, "/static/index.htm")
				return
			}
		}
		ctx.AbortWithStatus(404)
	})

	PreProcessGin(func(rail Rail, g *gin.Engine) {
		g.GET("/static/*filepath", func(c *gin.Context) {
			serveStaticFile(c, c.Param("filepath"))
		})
	})
}

func MustCompile(fs embed.FS, s string) *template.Template {
	tmplMapOnce.Do(func() { tmplMap = map[string]*template.Template{} })

	tmplMapMu.RLock()
	if t, ok := tmplMap[s]; ok {
		tmplMapMu.RUnlock()
		return t
	}
	tmplMapMu.RUnlock()

	tmplMapMu.Lock()
	defer tmplMapMu.Unlock()

	b, err := fs.ReadFile(s)
	if err != nil {
		panic(err)
	}

	t, err := template.New("").Parse(UnsafeByt2Str(b))
	if err != nil {
		panic(err)
	}
	tmplMap[s] = t
	Infof("Compiled template %v", s)
	return t
}
