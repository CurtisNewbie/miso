package miso

import (
	"embed"
	"html/template"
	"io"
	"sync"
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
