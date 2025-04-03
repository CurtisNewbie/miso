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

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"github.com/curtisnewbie/miso/version"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/dstutil"
)

const (
	MisoConfigPrefix = "misoconfig-"

	tagSection = "section"
	tagProp    = "prop"
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
)

func main() {
	flag.Usage = func() {
		util.Printlnf("\nmisoconfig - automatically generate configuration tables based on misoconfig-* comments\n")
		util.Printlnf("  Supported miso version: %v\n", version.Version)
		util.Printlnf("Usage of %s:", os.Args[0])
		flag.PrintDefaults()
		util.Printlnf("\nFor example:\n")
		util.Printlnf(`
// misoconfig-section: Web Server Configuration
const (

	// misoconfig-prop: enable http server | true
	PropServerEnabled = "server.enabled"

)`)
	}
	flag.Parse()

	files, err := walkDir(".", ".go")
	if err != nil {
		util.Printlnf("[ERROR] walkDir failed, %v", err)
		return
	}
	if err := parseFiles(files); err != nil {
		util.Printlnf("[ERROR] parseFiles failed, %v", err)
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
			util.Printlnf("[DEBUG] Found %v", f.Path)
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

	util.DebugPrintlnf(*Debug, "configs: %#v", configDecl)
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
					util.Printlnf("[DEBUG] parseMisoConfigTag() %v -> %v, command: %v, body: %v", path, s, pre, m)
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
		return nil, miso.WrapErr(err)
	}
	files := make([]FsFile, 0, len(entries))
	for _, et := range entries {
		fi, err := et.Info()
		if err != nil {
			util.Printlnf("[ERROR] %v", err)
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
			}
		}

		if !found {
			return section
		}

		for _, v := range n.Values {
			if bl, ok := v.(*dst.BasicLit); ok && bl.Kind == token.STRING {
				cd.Name = util.UnquoteStr(bl.Value)
			}
		}
		if cd.Name == "" {
			return section
		}
		util.DebugPrintlnf(*Debug, "parseConfigDecl() %v: (%v) %v -> %#v", srcPath, section, constName, cd)
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
		return util.ContainsAnyStr(n, "Common", "General")
	}
	sort.SliceStable(sections, func(i, j int) bool {
		if hasPrioritisedKw(sections[i].Name) {
			return true
		}
		return strings.Compare(sections[i].Name, sections[j].Name) < 0
	})

	// find file
	f, err := findConfigTableFile()
	if err != nil {
		util.Printlnf("Failed to find config table file, %v", err)
		return
	}
	if f == nil {
		util.Printlnf("Failed to find config table file")
		return
	}
	defer f.Close()

	sb := util.SLPinter{}

	for _, sec := range sections {
		if len(sec.Configs) < 1 {
			continue
		}
		maxNameLen := len("property")
		maxDescLen := len("description")
		maxValLen := len("default value")
		for _, c := range sec.Configs {
			if len(c.Name) > maxNameLen {
				maxNameLen = len(c.Name)
			}
			if len(c.Description) > maxDescLen {
				maxDescLen = len(c.Description)
			}
			if len(c.DefaultValue) > maxValLen {
				maxValLen = len(c.DefaultValue)
			}
		}

		sb.Printlnf("\n## %v\n", sec.Name)
		sb.Println(util.NamedSprintf("| ${Name} | ${Description} | ${DefaultValue} |", map[string]any{
			"Name":         util.PadSpace(-maxNameLen, "property"),
			"Description":  util.PadSpace(-maxDescLen, "description"),
			"DefaultValue": util.PadSpace(-maxValLen, "default value"),
		}))
		sb.Println(util.NamedSprintf("| ${Name} | ${Description} | ${DefaultValue} |", map[string]any{
			"Name":         util.PadToken(-maxNameLen, "---", "-"),
			"Description":  util.PadToken(-maxDescLen, "---", "-"),
			"DefaultValue": util.PadToken(-maxValLen, "---", "-"),
		}))
		for _, c := range sec.Configs {
			c.Name = util.PadSpace(-maxNameLen, c.Name)
			c.Description = util.PadSpace(-maxDescLen, c.Description)
			c.DefaultValue = util.PadSpace(-maxValLen, c.DefaultValue)
			sb.Println(util.NamedSprintfv("| ${Name} | ${Description} | ${DefaultValue} |", c))
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
		util.Printlnf("Failed to write config table file: %v, %v", f.Name(), err)
	} else {
		util.Printlnf("Generated config table to %v", f.Name())
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
		util.DebugPrintlnf(*Debug, "path: %v, pkg: %v", path, pkg)

		f, err := util.ReadWriteFile(path)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		n := 0
		for _, c := range src {
			if c.DefaultValue == "" {
				continue
			}
			n++
		}
		if n < 1 {
			continue
		}

		b := strings.Builder{}
		b.WriteString("func init() {")
		for _, c := range src {
			if c.DefaultValue == "" {
				continue
			}
			dv := c.DefaultValue
			dvLower := strings.ToLower(dv)
			if dvLower == "true" || dvLower == "false" || digits.MatchString(dv) {
				// bool or int
			} else if codeBlock.MatchString(dv) {
				// code block
				vv := codeBlock.FindAllStringSubmatch(dv, 1)[0][1]
				dv = vv
			} else {
				dv = "\"" + dv + "\""
			}
			if pkg == "miso" {
				b.WriteString("\n\t" + fmt.Sprintf("SetDefProp(%v, %v)", c.ConstName, dv))
			} else {
				b.WriteString("\n\t" + fmt.Sprintf("miso.SetDefProp(%v, %v)", c.ConstName, dv))
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
		util.Printlnf("Generated default config code in %v", f.Name())
	}
}

func findConfigTableFile() (*os.File, error) {
	if *Path != "" {
		return util.ReadWriteFile(*Path)
	}

	if err := util.MkdirAll("./doc"); err != nil {
		return nil, err
	}

	files, err := walkDir("./doc", ".md")
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if f.File.Name() == DefaultConfigurationFileName {
			return util.ReadWriteFile(f.Path)
		}
	}
	return util.ReadWriteFile("./doc/" + DefaultConfigurationFileName)
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
