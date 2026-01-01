package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/curtisnewbie/miso/errs"
	"github.com/curtisnewbie/miso/tools"
	"github.com/curtisnewbie/miso/util/async"
	"github.com/curtisnewbie/miso/util/cli"
	"github.com/curtisnewbie/miso/util/flags"
	"github.com/curtisnewbie/miso/util/osutil"
	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/curtisnewbie/miso/util/strutil"
	"github.com/curtisnewbie/miso/version"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/dstutil"
)

const (
	MisoConfigPrefix = "misoconfig-"

	tagSection = "section"
	tagProp    = "prop"
	tagAlias   = "alias"
	tagDocOnly = "doc-only"
)

var (
	digits    = regexp.MustCompile(`^[0-9]*$`)
	codeBlock = regexp.MustCompile("^`(.*)`$")
)

const (
	DefaultConfigurationFileName = "config.md"
	ConfigTableEmbedStart        = "<!-- misoconfig-table-start -->"
	ConfigTableEmbedEnd          = "<!-- misoconfig-table-end -->"
	ConfigDefaultEmbedStart      = "// misoconfig-default-start"
	ConfigDefaultEmbedEnd        = "// misoconfig-default-end"
)

var (
	Debug = flag.Bool("debug", false, "Enable debug log")
	Path  = flag.String("path", "", "Path to the generated markdown config table file")

	log = cli.NewLog(cli.LogWithDebug(Debug), cli.LogWithCaller(func(level string) bool { return level != "INFO" }))
)

func main() {
	flags.WithDescriptionBuilder(func(printlnf func(v string, args ...any)) {
		printlnf("misoconfig - automatically generate configuration tables based on misoconfig-* comments\n")
		printlnf("  Supported miso version: %v\n", version.Version)
	})
	flags.WithExtraBuilder(func(printlnf func(v string, args ...any)) {
		printlnf("\nFor example:")
		printlnf(`
In prop.go:

  // misoconfig-section: Web Server Configuration
  const (

	  // misoconfig-prop: enable http server | true
	  PropServerEnabled = "server.enabled"

	  // misoconfig-prop: my prop
	  // misoconfig-alias: old-prop
	  PropDeprecated = "new-prop"

	  // misoconfig-prop: my special prop
	  // misoconfig-doc-only
	  PropDocOnly = "prod-only-shown-in-doc"

	  // misoconfig-default-start
	  // misoconfig-default-end
  )

In ./doc/config.md:

  <!-- misoconfig-table-start -->
  <!-- misoconfig-table-end -->
`)
	})
	flags.Parse()

	files, err := walkDir(".", ".go")
	if err != nil {
		log.Errorf("walkDir failed, %v", err)
		return
	}
	if err := parseFiles(files); err != nil {
		log.Errorf("parseFiles failed, %v", err)
	}
}

type FsFile struct {
	Path string
	File fs.FileInfo
}

func parseFiles(files []FsFile) error {
	dstFiles, err := parseFileAst(files)
	if err != nil {
		return err
	}

	if *Debug {
		for _, f := range dstFiles {
			log.Debugf("Found %v", f.Path)
		}
	}

	configDecl := map[string][]ConfigDecl{}
	var section string
	for _, df := range dstFiles {
		dstutil.Apply(df.Dst,
			func(c *dstutil.Cursor) bool {
				// parse config declaration
				if ns := parseConfigDecl(c, df, section, configDecl); ns != "" {
					section = ns
				}
				return true
			},
			func(cursor *dstutil.Cursor) bool {
				return true
			},
		)
	}

	log.Debugf("configs: %#v", configDecl)
	flushConfigTable(configDecl)
	return nil
}

type DstFile struct {
	Dst  *dst.File
	Path string
}

type Pair struct {
	K string
	V string
}

type MisoConfigTag struct {
	Command string
	Body    string
}

func (m *MisoConfigTag) BodyKV() (Pair, bool) {
	return m.BodyKVTok(":")
}

func (m *MisoConfigTag) BodyKVTok(tok string) (Pair, bool) {
	i := strings.Index(m.Body, tok)
	if i < 0 {
		return Pair{K: m.Body}, false
	}
	return Pair{
		K: strings.TrimSpace(m.Body[:i]),
		V: strings.TrimSpace(m.Body[i+1:]),
	}, true
}

func parseMisoConfigTag(path string, start dst.Decorations) ([]MisoConfigTag, bool) {
	t := []MisoConfigTag{}
	for _, s := range start {
		s = strings.TrimSpace(s)
		s, _ = strings.CutPrefix(s, "//")
		s = strings.TrimSpace(s)
		s, _ = strings.CutPrefix(s, "-")
		s = strings.TrimSpace(s)
		if m, ok := strings.CutPrefix(s, MisoConfigPrefix); ok { // e.g., misoconfig-prop
			if pi := strings.Index(m, ":"); pi > -1 { // e.g., "misoconfig-prop: ..."
				pre := m[:pi]
				m = m[pi+1:]
				if *Debug {
					log.Debugf("parseMisoConfigTag() %v -> %v, command: %v, body: %v", path, s, pre, m)
				}
				pre = strings.TrimSpace(pre)
				t = append(t, MisoConfigTag{
					Command: pre,
					Body:    strings.TrimSpace(m),
				})
			} else { // e.g., "misoconfig-section"
				trimmed := strings.TrimSpace(m)
				t = append(t, MisoConfigTag{
					Command: trimmed,
					Body:    trimmed,
				})
				continue
			}
		}
	}
	return t, len(t) > 0
}

func parseFileAst(files []FsFile) ([]DstFile, error) {
	parsed := make([]DstFile, 0)
	for _, f := range files {
		p := f.Path
		if path.Base(p) == "misoapi_generated.go" {
			continue
		}
		d, err := decorator.ParseFile(nil, p, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, DstFile{
			Dst:  d,
			Path: p,
		})
	}
	return parsed, nil
}

func walkDir(n string, suffix string) ([]FsFile, error) {
	entries, err := os.ReadDir(n)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	files := make([]FsFile, 0, len(entries))
	for _, et := range entries {
		fi, err := et.Info()
		if err != nil {
			log.Errorf("%v", err)
			continue
		}
		p := n + "/" + fi.Name()
		if et.IsDir() {
			ff, err := walkDir(p, suffix)
			if err == nil {
				files = append(files, ff...)
			}
		} else {
			if strings.HasSuffix(fi.Name(), suffix) {
				files = append(files, FsFile{File: fi, Path: p})
			}
		}
	}
	return files, nil
}

type ConfigSection struct {
	Name    string
	Configs []ConfigDecl
}
type ConfigDecl struct {
	Source       string
	Package      string
	Name         string
	ConstName    string
	Description  string
	DefaultValue string
	Alias        string
	AliasSince   string
	DocOnly      bool
}

func parseConfigDecl(cursor *dstutil.Cursor, df DstFile, section string, configs map[string][]ConfigDecl) (newSection string) {
	srcPath := df.Path
	pkg := df.Dst.Name

	switch n := cursor.Node().(type) {
	case *dst.GenDecl:
		comment := n.Decs.Start
		tags, ok := parseMisoConfigTag(srcPath, comment)
		if !ok {
			return section
		}
		for _, t := range tags {
			if t.Command == tagSection {
				section = t.Body
			}
		}
	case *dst.ValueSpec:
		comment := n.Decs.Start
		tags, ok := parseMisoConfigTag(srcPath, comment)
		if !ok {
			return section
		}

		var constName string
		for _, n := range n.Names {
			constName = n.Name
		}

		var found bool = false
		var cd ConfigDecl = ConfigDecl{Source: srcPath, Package: pkg.Name, ConstName: constName}
		for _, t := range tags {
			switch t.Command {
			case tagProp:
				found = true
				p, _ := t.BodyKVTok("|")
				cd.Description = p.K
				cd.DefaultValue = p.V
			case tagAlias:
				found = true
				p, _ := t.BodyKVTok("|")
				cd.Alias = p.K
				cd.AliasSince = p.V
			case tagDocOnly:
				cd.DocOnly = true
			}
		}

		if !found {
			return section
		}

		for _, v := range n.Values {
			if bl, ok := v.(*dst.BasicLit); ok && bl.Kind == token.STRING {
				cd.Name = strutil.UnquoteStr(bl.Value)
			}
		}
		if cd.Name == "" {
			return section
		}
		log.Debugf("%v: (%v) %v -> %#v", srcPath, section, constName, cd)
		sec := section
		if sec == "" {
			sec = "General"
		}
		configs[sec] = append(configs[sec], cd)
	}
	return section
}

func flushConfigTable(configs map[string][]ConfigDecl) {
	if len(configs) < 1 {
		return
	}

	sections := make([]ConfigSection, 0, len(configs))
	for k, v := range configs {
		sections = append(sections, ConfigSection{Configs: v, Name: k})
	}
	hasPrioritisedKw := func(n string) bool {
		return strutil.ContainsAnyStr(n, "Common", "General")
	}
	sort.SliceStable(sections, func(i, j int) bool {
		if hasPrioritisedKw(sections[i].Name) {
			return true
		} else if hasPrioritisedKw(sections[j].Name) {
			return false
		}
		return strings.Compare(sections[i].Name, sections[j].Name) < 0
	})

	// find file
	f, err := findConfigTableFile()
	if err != nil {
		log.Infof("Failed to find config table file, %v", err)
		return
	}
	if f == nil {
		log.Infof("Failed to find config table file")
		return
	}
	defer f.Close()

	sb := strutil.SLPinter{}
	wlen := func(s string) int { return strutil.StrWidth(s) }

	for _, sec := range sections {
		if len(sec.Configs) < 1 {
			continue
		}
		maxNameLen := wlen("property")
		maxDescLen := wlen("description")
		maxValLen := wlen("default value")
		configs := slutil.CopyFilter(sec.Configs, func(c ConfigDecl) bool { return c.Description != "" })
		for _, c := range configs {
			nameLen := wlen(c.Name)
			descLen := wlen(c.Description)
			defValLen := wlen(c.DefaultValue)
			if nameLen > maxNameLen {
				maxNameLen = nameLen
			}
			if descLen > maxDescLen {
				maxDescLen = descLen
			}
			if defValLen > maxValLen {
				maxValLen = defValLen
			}
		}

		sb.Printlnf("\n## %v\n", sec.Name)
		sb.Println(strutil.NamedSprintf("| ${Name} | ${Description} | ${DefaultValue} |", map[string]any{
			"Name":         strutil.PadSpace(-maxNameLen, "property"),
			"Description":  strutil.PadSpace(-maxDescLen, "description"),
			"DefaultValue": strutil.PadSpace(-maxValLen, "default value"),
		}))
		sb.Println(strutil.NamedSprintf("| ${Name} | ${Description} | ${DefaultValue} |", map[string]any{
			"Name":         strutil.PadToken(-maxNameLen, "---", "-"),
			"Description":  strutil.PadToken(-maxDescLen, "---", "-"),
			"DefaultValue": strutil.PadToken(-maxValLen, "---", "-"),
		}))
		for _, c := range configs {
			c.Name = strutil.PadSpace(-maxNameLen, c.Name)
			c.Description = strutil.PadSpace(-maxDescLen, c.Description)
			c.DefaultValue = strutil.PadSpace(-maxValLen, c.DefaultValue)
			sb.Println(strutil.NamedSprintfv("| ${Name} | ${Description} | ${DefaultValue} |", c))
		}
	}

	// check if we are embedding config table or replacing the whole content
	out := sb.String()
	doEmbed := false

	content, err := io.ReadAll(f)
	if err == nil {
		contents := string(content)
		v, embed := parseEmbed(contents, out, ConfigTableEmbedStart, ConfigTableEmbedEnd)
		if embed {
			out = v
			doEmbed = true
		}
	}

	if !doEmbed {
		out = "# Configurations\n\n" + "For more configuration, see [github.com/curtisnewbie/miso](https://github.com/CurtisNewbie/miso/blob/main/doc/config.md).\n" + out
	}

	f.Seek(0, io.SeekStart)
	f.Truncate(0)
	if _, err := f.WriteString(out); err != nil {
		log.Infof("Failed to write config table file: %v, %v", f.Name(), err)
	} else {
		log.Infof("Generated config table to %v", f.Name())
	}

	// write default value in golang source code
	srcMap := map[string][]ConfigDecl{}
	for _, v := range sections {
		for _, c := range v.Configs {
			srcMap[c.Source] = append(srcMap[c.Source], c)
		}
	}

	for _, src := range srcMap {
		if len(src) < 1 {
			continue
		}
		path := src[0].Source
		pkg := src[0].Package
		log.Debugf("path: %v, pkg: %v", path, pkg)

		f, err := osutil.OpenRWFile(path)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		skipConfig := func(c ConfigDecl) bool { return (c.DefaultValue == "" && c.Alias == "") || c.DocOnly }
		for _, c := range src {
			if skipConfig(c) {
				continue
			}
		}

		b := strings.Builder{}
		b.WriteString("func init() {")

		anyAlias := false
		for _, c := range src {
			if skipConfig(c) {
				continue
			}
			if c.Alias != "" {
				anyAlias = true
				break
			}
		}

		// viper's alias doesn't really work when we load yaml content, we have to give up the alias feature.
		if anyAlias {
			var pkgPrefix = ""
			if pkg != "miso" {
				pkgPrefix = "miso."
			}
			iw := strutil.NewIndentWriter("\t")
			iw.SetIndent(1)
			iw.Writef("%vPostServerBootstrap(func(rail %vRail) error {", pkgPrefix, pkgPrefix)
			iw.StepIn(func(iw *strutil.IndentWriter) {
				iw.Writef("deprecatedProps := [][]string{}")
				for _, c := range src {
					if skipConfig(c) {
						continue
					}
					if c.Alias != "" {
						iw.Writef("deprecatedProps = append(deprecatedProps, []string{\"%v\", \"%v\", %v})", c.Alias, c.AliasSince, c.ConstName)
					}
				}

				iw.Writef("for _, p := range deprecatedProps {")
				iw.StepIn(func(iw *strutil.IndentWriter) {
					iw.Writef("if %vHasProp(p[0]) {", pkgPrefix)
					iw.StepIn(func(iw *strutil.IndentWriter) {
						iw.Writef("%vErrorf(\"Config prop: '%%v' has been deprecated since '%%v', please change to '%%v'\", p[0], p[1], p[2])", pkgPrefix)
					})
					iw.Writef("}")
				})
				iw.Writef("}")
				iw.Writef("return nil")
			})
			iw.Writef("})")
			b.WriteString("\n" + iw.String())
		}

		// SetDefProp(...)
		for _, c := range src {
			if skipConfig(c) {
				continue
			}

			var pkgPrefix = ""
			if pkg != "miso" {
				pkgPrefix = "miso."
			}
			if c.DefaultValue != "" {
				dv := c.DefaultValue
				dvLower := strings.ToLower(dv)
				if dvLower == "true" || dvLower == "false" || digits.MatchString(dv) {
					// bool or int
				} else if codeBlock.MatchString(dv) {
					// code block
					vv := codeBlock.FindAllStringSubmatch(dv, 1)[0][1]
					dv = vv
				} else {
					rdv := []rune(dv)
					if len(rdv) < 1 || rdv[0] != '"' || rdv[len(rdv)-1] != '"' {
						dv = "\"" + dv + "\""
					}
				}
				b.WriteString("\n\t" + fmt.Sprintf("%vSetDefProp(%v, %v)", pkgPrefix, c.ConstName, dv))
			}

		}

		b.WriteString("\n}")

		buf, err := io.ReadAll(f)
		if err != nil {
			panic(err)
		}
		content := string(buf)
		v, doEmbed := parseEmbed(content, b.String(), ConfigDefaultEmbedStart, ConfigDefaultEmbedEnd)
		if !doEmbed {
			continue
		}
		content = v
		f.Truncate(0)
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			panic(err)
		}
		if _, err := f.WriteString(content); err != nil {
			panic(err)
		}

		log.Infof("Generated default config code in %v", f.Name())
		f.Close()
		async.PanicSafeRun(func() { tools.RunGoImports(path) }) // fix imports
	}
}

func findConfigTableFile() (*os.File, error) {
	if *Path != "" {
		return osutil.OpenRWFile(*Path)
	}

	if err := osutil.MkdirAll("./doc"); err != nil {
		return nil, err
	}

	files, err := walkDir("./doc", ".md")
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if f.File.Name() == DefaultConfigurationFileName {
			return osutil.OpenRWFile(f.Path)
		}
	}
	return osutil.OpenRWFile("./doc/" + DefaultConfigurationFileName)
}

func parseEmbed(contents string, embedded string, start string, end string) (string, bool) {
	startOffset, endOffset := -1, -1
	lines := strings.Split(contents, "\n")
	for i, l := range lines {
		switch strings.TrimSpace(l) {
		case start:
			startOffset = i
		case end:
			endOffset = i
		}
	}
	if startOffset > -1 && endOffset > -1 {
		before := strings.Join(lines[:startOffset+1], "\n")
		after := strings.Join(lines[endOffset:], "\n")
		return before + "\n" + embedded + "\n\n" + after, true
	}
	return "", false
}
