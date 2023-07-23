package protopkg

import (
	"fmt"
	"sort"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const (
	protoPkgPackageRuleKind = "protopkg_package"
)

type protoPkgPackageConfig struct {
	name  string
	label label.Label
}

func newProtoPkgPackageConfig(name string) *protoPkgPackageConfig {
	return &protoPkgPackageConfig{name: name}
}

// parseDirectives is called in each directory visited by gazelle.  The relative
// directory name is given by 'rel' and the list of directives in the BUILD file
// are specified by 'directives'.
func (p *protoPkgPackageConfig) parseDirective(c *protoPkgConfig, name, param, value string) (err error) {
	switch param {
	case "repo":
		p.label.Repo = value
	case "package":
		p.label.Pkg = value
	case "name":
		p.label.Name = value
	default:
		return fmt.Errorf("unknown directive attribute %s.%s", protoPkgPackageDirective, param)
	}
	return nil
}

type protoPkgPackage struct {
	cfg *protoPkgPackageConfig
}

func newProtoPkgPackage(cfg *protoPkgPackageConfig) *protoPkgPackage {
	return &protoPkgPackage{cfg: cfg}
}

func (p *protoPkgPackage) generateRule(resolver *resolver) *rule.Rule {
	provided := resolver.Provided("protobuf", "protopkg_file")
	labels := make([]label.Label, 0, len(provided))
	for k := range provided {
		if p.cfg.label.Repo != "" {
			if k.Repo != p.cfg.label.Repo {
				continue
			}
		}
		if p.cfg.label.Pkg != "" {
			if k.Pkg != p.cfg.label.Pkg {
				continue
			}
		}
		if p.cfg.label.Name != "" {
			if k.Name != p.cfg.label.Name {
				continue
			}
		}
		labels = append(labels, k)
	}

	deps := make([]string, len(labels))
	for i, label := range labels {
		deps[i] = label.String()
	}
	sort.Strings(deps)

	r := rule.NewRule(protoPkgPackageRuleKind, p.cfg.name)
	r.SetAttr("deps", deps)
	return r
}
