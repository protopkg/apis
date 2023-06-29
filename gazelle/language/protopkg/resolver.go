package protopkg

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

// importLabels records which labels are associated with a given proto import
// statement.
type importLabels map[string][]label.Label

func newResolver() *resolver {
	return &resolver{
		known: make(map[string]importLabels),
	}
}

// resolver implements ImportResolver.
type resolver struct {
	// known is a mapping between lang and importLabel map
	known map[string]importLabels
}

// LoadFile reads a protoresolve csv file.
func (r *resolver) LoadFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return r.Load(f)
}

// Load reads input and returns a list of items.  Comment lines beginning
// with '#' are ignored.
func (r *resolver) Load(in io.Reader) error {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ",", 4)
		if len(parts) != 4 {
			log.Printf("warn: parse %q, expected 4 items, got %d", line, len(parts))
			continue
		}
		lang := parts[0]
		impLang := parts[1]
		imp := parts[2]
		lbl, err := label.Parse(parts[3])
		if err != nil {
			return fmt.Errorf("malformed label at position 4 in %s: %v", line, err)
		}
		r.Provide(lang, impLang, imp, lbl)
	}
	return nil
}

// Resolve implements part of the ImportResolver interface.
func (r *resolver) Resolve(lang, impLang, imp string) []resolve.FindResult {
	key := langKey(lang, impLang)
	known := r.known[key]
	if known == nil {
		known = r.known[lang]
	}
	if known == nil {
		return nil
	}
	if got, ok := known[imp]; ok {
		res := make([]resolve.FindResult, len(got))
		// use last-wins semantics by providing FindResults in reverse to how
		// they were provided.
		var index int
		for i := len(got) - 1; i >= 0; i-- {
			res[index] = resolve.FindResult{Label: got[i]}
			index++
		}
		return res
	}
	return nil
}

// Provide implements part of the ImportResolver interface.
func (r *resolver) Provide(lang, impLang, imp string, from label.Label) {
	key := langKey(lang, impLang)
	known, ok := r.known[key]
	if !ok {
		known = make(map[string][]label.Label)
		r.known[key] = known
	}
	for _, v := range known[imp] {
		if v == from {
			return
		}
	}
	known[imp] = append(known[imp], from)
}

// Provided implements the ImportProvider interface.
func (r *resolver) Provided(lang, impLang string) map[label.Label][]string {
	if len(r.known) == 0 {
		return nil
	}
	result := make(map[label.Label][]string)
	key := langKey(lang, impLang)
	known := r.known[key]
	if known == nil {
		known = r.known[lang]
	}
	if known == nil {
		return nil
	}
	for imp, ll := range known {
		for _, l := range ll {
			result[l] = append(result[l], imp)
		}
	}
	return result
}

// Imports implements part of the ImportResolver interface.
func (r *resolver) Imports(lang, impLang string, visitor func(imp string, location []label.Label) bool) {
	key := langKey(lang, impLang)
	known := r.known[key]
	if known == nil {
		known = r.known[lang]
	}
	if known == nil {
		return
	}
	for k, v := range known {
		if !visitor(k, v) {
			break
		}
	}
}

// overrideSpec is a copy of the same private type in resolve/config.go.  It must be
// kept in sync with the original to avoid discrepancy with the expected memory
// layout.
type overrideSpec struct {
	imp  resolve.ImportSpec
	lang string
	dep  label.Label
}

func langKey(lang, impLang string) string {
	return lang + " " + impLang
}

func keyLang(key string) (string, string) {
	parts := strings.SplitN(key, " ", 2)
	return parts[0], parts[1]
}
