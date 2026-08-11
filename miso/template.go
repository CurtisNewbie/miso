package miso

import (
	"embed"
	"html/template"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/curtisnewbie/miso/util/strutil"
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
//   - fs: the embedded filesystem containing the static files.
//   - dir: the directory in the embedded filesystem where the static files are located.
//   - hostPrefix: optional prefix that the app is hosted under (e.g. when it's mounted behind a gateway),
//     it's prepended to the redirect URLs, must NOT be included in the actual route paths.
//
// Notice that index.html must be renamed to index.htm or else it won't work.
//
// If you are using Angular framework, you may add extra build param as follows. The idea is still the same for other frameworks.
//
//	ng build --baseHref=/static/
func PrepareWebStaticFs(fs embed.FS, dir string, hostPrefix ...string) {
	PrepareWebStaticFsWithPrefix(fs, dir, "/static", hostPrefix...)
}

// PrepareWebStaticFsWithPrefix prepares embedded static files under the given urlPrefix.
//
//   - fs: the embedded filesystem containing the static files.
//   - dir: the directory in the embedded filesystem where the static files are located.
//   - urlPrefix: the url prefix under which the static files are served, e.g. '/static', leading/trailing slashes are normalized.
//   - hostPrefix: optional prefix that the app is hosted under (e.g. when it's mounted behind a gateway),
//     it's prepended to the redirect URLs, must NOT be included in the actual route paths.
//
// Notice that index.html must be renamed to index.htm or else it won't work.
func PrepareWebStaticFsWithPrefix(fs embed.FS, dir string, urlPrefix string, hostPrefix ...string) {
	urlPrefix = normalizeWebStaticPrefix(urlPrefix)

	serveStaticFile := func(c *gin.Context, fp string) {
		Debugf("Serving static file: %v", fp)
		c.FileFromFS(path.Join(dir, fp), http.FS(fs))
	}

	var host string
	if v, ok := slutil.First(hostPrefix); ok {
		host = normalizeWebStaticPrefix(v)
	}

	setNoRouteHandler(func(ctx *gin.Context, rail Rail) {
		// why are we using index.htm instead of index.html.
		//
		// https://stackoverflow.com/questions/69462376/serving-react-static-files-in-golang-gin-gonic-using-goembed-giving-404-error-o
		// https://cs.opensource.google/go/go/+/refs/tags/go1.21.5:src/net/http/fs.go;l=604
		// https://github.com/gin-contrib/static/issues/19
		redirectURL := host + urlPrefix + "/index.htm"
		if ctx.Request.Method == http.MethodGet {
			requestPath := ctx.Request.URL.Path
			if requestPath == "/" || hasWebStaticPrefix(requestPath, urlPrefix) {
				ctx.Redirect(http.StatusTemporaryRedirect, redirectURL)
				return
			}
		}
		ctx.AbortWithStatus(404)
	})

	BeforeWebRouteRegister(func(rail Rail) error {
		HttpGet(urlPrefix+"/*filepath", RawHandler(func(inb *Inbound) {
			c := inb.Engine().(*gin.Context)
			cp := c.Param("filepath")
			// empty path or path ending with '/' resolves to a directory,
			// serving it via FileFromFS would cause redirect loops (e.g. /internal/web/ -> ./),
			// fall back to index.htm instead
			if cp == "" || strings.HasSuffix(cp, "/") {
				cp = "index.htm"
			}
			serveStaticFile(c, cp)
		}))
		return nil
	})
}

func normalizeWebStaticPrefix(prefix string) string {
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return ""
	}
	return "/" + prefix
}

func hasWebStaticPrefix(requestPath string, urlPrefix string) bool {
	if urlPrefix == "" {
		return true
	}
	return requestPath == urlPrefix || strings.HasPrefix(requestPath, urlPrefix+"/")
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

	t, err := template.New("").Parse(strutil.UnsafeByt2Str(b))
	if err != nil {
		panic(err)
	}
	tmplMap[s] = t
	Infof("Compiled template %v", s)
	return t
}
