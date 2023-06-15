package protopkg

import "github.com/bazelbuild/bazel-gazelle/language"

// NewLanguage is called by Gazelle to install this language extension in a
// binary.  This package "language/protobuf" is separate from
// "pkg/language/protobuf" because this one bundles all the plugin and rule
// implementations in the repo whereas the other is the "pure" language
// implementation, with no pre-population of the registries.
func NewLanguage() language.Language {
	return &protopkgLang{}
}
