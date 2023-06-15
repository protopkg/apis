package protopkg

import (
	"fmt"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/stackb/rules_proto/pkg/protoc"
)

const (
	protoPkgApisName    = "protopkg_apis"
	protoPkgLibraryName = "protopkg_library"
)

func init() {
	protoc.Rules().MustRegisterRule("protopkg:apis:"+protoPkgLibraryName, &protoPkgLibrary{})
}

// protoPkgLibrary implements LanguageRule for the 'protopkg_library'.
// rule from @protopkg_apis.
type protoPkgLibrary struct{}

// Name implements part of the LanguageRule interface.
func (s *protoPkgLibrary) Name() string {
	return protoPkgLibraryName
}

// KindInfo implements part of the LanguageRule interface.
func (s *protoPkgLibrary) KindInfo() rule.KindInfo {
	return rule.KindInfo{
		MergeableAttrs: map[string]bool{
			"deps": true,
		},
	}
}

// LoadInfo implements part of the LanguageRule interface.
func (s *protoPkgLibrary) LoadInfo() rule.LoadInfo {
	return rule.LoadInfo{
		Name:    fmt.Sprintf("@%s//rules:%s.bzl", protoPkgApisName, protoPkgLibraryName),
		Symbols: []string{protoPkgLibraryName},
	}
}

// ProvideRule implements part of the LanguageRule interface.
func (s *protoPkgLibrary) ProvideRule(cfg *protoc.LanguageRuleConfig, config *protoc.ProtocConfiguration) protoc.RuleProvider {
	return &protoPkgLibraryRule{ruleConfig: cfg, config: config}
}

// protoPkgLibrary implements RuleProvider for the 'proto_compile' rule.
type protoPkgLibraryRule struct {
	config     *protoc.ProtocConfiguration
	ruleConfig *protoc.LanguageRuleConfig
}

// Kind implements part of the ruleProvider interface.
func (s *protoPkgLibraryRule) Kind() string {
	return protoPkgLibraryName
}

// Name implements part of the ruleProvider interface.
func (s *protoPkgLibraryRule) Name() string {
	return fmt.Sprintf("%s_pkg", s.config.Library.BaseName())
}

// Visibility provides visibility labels.
func (s *protoPkgLibraryRule) Visibility() []string {
	return s.ruleConfig.GetVisibility()
}

// Rule implements part of the ruleProvider interface.
func (s *protoPkgLibraryRule) Rule(otherGen ...*rule.Rule) *rule.Rule {
	newRule := rule.NewRule(s.Kind(), s.Name())

	newRule.SetAttr("deps", []string{s.config.Library.Name()})

	visibility := s.Visibility()
	if len(visibility) > 0 {
		newRule.SetAttr("visibility", visibility)
	}

	return newRule
}

// Imports implements part of the RuleProvider interface.
func (s *protoPkgLibraryRule) Imports(c *config.Config, r *rule.Rule, file *rule.File) []resolve.ImportSpec {
	return nil
}

// Resolve implements part of the RuleProvider interface.
func (s *protoPkgLibraryRule) Resolve(c *config.Config, ix *resolve.RuleIndex, r *rule.Rule, imports []string, from label.Label) {
}
