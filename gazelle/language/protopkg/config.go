package protopkg

import (
	"fmt"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const (
	protoPkgPackageDirective = "protopkg_package"
)

// getOrCreateprotoPkgConfig either inserts a new config into the map under the
// language name or replaces it with a clone.
func (l *protoPkgLanguage) getOrCreateProtoPkgConfig(config *config.Config) *protoPkgConfig {
	var cfg *protoPkgConfig
	if existingExt, ok := config.Exts[protoPkgLanguageName]; ok {
		cfg = existingExt.(*protoPkgConfig).Clone()
	} else {
		cfg = newProtoPkgConfig(config)
	}
	config.Exts[protoPkgLanguageName] = cfg
	return cfg
}

// getProtoPkgConfig returns the associated package config.
func getProtoPkgConfig(config *config.Config) *protoPkgConfig {
	if cfg, ok := config.Exts[protoPkgLanguageName].(*protoPkgConfig); ok {
		return cfg
	}
	return nil
}

// protoPkgConfig represents the config extension for the protobuf language.
type protoPkgConfig struct {
	// config is the parent gazelle config.
	config *config.Config
	// pkgs is the set of configurations for a protopkg_package rule.  If the
	// this configured, it marks the package (BUILD.bazel file) to which we
	// should be generating a protopkg_package rule.  Since this is a marker
	// rather than a property that should be inherited to all other build files,
	// it is not cloned.
	pkgs map[string]*protoPkgPackageConfig
}

// newProtoPkgConfig initializes a new protoPkgConfig.
func newProtoPkgConfig(config *config.Config) *protoPkgConfig {
	return &protoPkgConfig{
		config: config,
		pkgs:   map[string]*protoPkgPackageConfig{},
	}
}

// Clone copies this config to a new one.
func (c *protoPkgConfig) Clone() *protoPkgConfig {
	clone := newProtoPkgConfig(c.config)
	return clone
}

// ParseDirectives is called in each directory visited by gazelle.  The relative
// directory name is given by 'rel' and the list of directives in the BUILD file
// are specified by 'directives'.
func (c *protoPkgConfig) ParseDirectives(rel string, directives []rule.Directive) (err error) {
	// log.Printf("parsing directives: %s: %+v", rel, directives)
	for _, d := range directives {
		switch d.Key {
		case protoPkgPackageDirective:
			err = c.parseProtoPackageRepositoryDirective(d)
		}
		if err != nil {
			return fmt.Errorf("parse %v: %w", d, err)
		}
	}
	return
}

func (c *protoPkgConfig) parseProtoPackageRepositoryDirective(d rule.Directive) error {
	fields := strings.Fields(d.Value)
	if len(fields) != 3 {
		return fmt.Errorf("invalid directive %v: expected three fields, got %d", d, len(fields))
	}
	name, param, value := fields[0], fields[1], fields[2]
	pkg, ok := c.pkgs[name]
	if !ok {
		pkg = newProtoPkgPackageConfig(name)
		c.pkgs[name] = pkg
	}
	return pkg.parseDirective(c, name, param, value)
}

func (c *protoPkgConfig) rules(resolver *resolver) (rules []*rule.Rule) {
	for _, pkg := range c.pkgs {
		gen := newProtoPkgPackage(pkg)
		rules = append(rules, gen.generateRule(resolver))
	}
	return
}

func (c *protoPkgConfig) empty() (rules []*rule.Rule) {
	return
}
