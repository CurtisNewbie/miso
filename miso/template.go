package miso

import (
	"embed"
	"html/template"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/curtisnewbie/miso/util"
	"github.com/curtisnewbie/miso/util/slutil"
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
// Static files are all served by paths with prefix '/static'.
//
// Notice that index.html must be renamed to index.htm or else it won't work.
//
// If you are using Angular framework, you may add extra build param as follows. The idea is still the same for other frameworks.
//
//	ng build --baseHref=/static/
func PrepareWebStaticFs(fs embed.FS, dir string, hostPrefix ...string) {
	serveStaticFile := func(c *gin.Context, fp string) {
		Debugf("Serving static file: %v", fp)
		c.FileFromFS(path.Join(dir, fp), http.FS(fs))
	}

	var hp string
	if v, ok := slutil.SliceFirst(hostPrefix); ok {
		hp = v
	}
	if hp != "" {
		hp = "/" + hp
	}

	setNoRouteHandler(func(ctx *gin.Context, rail Rail) {
		// why are we using index.htm instead of index.html.
		//
		// https://stackoverflow.com/questions/69462376/serving-react-static-files-in-golang-gin-gonic-using-goembed-giving-404-error-o
		// https://cs.opensource.google/go/go/+/refs/tags/go1.21.5:src/net/http/fs.go;l=604
		// https://github.com/gin-contrib/static/issues/19
		if ctx.Request.Method == "GET" {
			if ctx.Request.RequestURI == "/" || strings.HasPrefix(ctx.Request.RequestURI, "/static") {
				ctx.Redirect(http.StatusTemporaryRedirect, hp+"/static/index.htm")
				return
			}
		}
		ctx.AbortWithStatus(404)
	})

	BeforeWebRouteRegister(func(rail Rail) error {
		HttpGet("/static/*filepath", RawHandler(func(inb *Inbound) {
			c := inb.Engine().(*gin.Context)
			cp := c.Param("filepath")
			if cp == "" {
				cp = "index.htm"
			}
			serveStaticFile(c, cp)
		}))
		return nil
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

	t, err := template.New("").Parse(util.UnsafeByt2Str(b))
	if err != nil {
		panic(err)
	}
	tmplMap[s] = t
	Infof("Compiled template %v", s)
	return t
}
