package main

// ProtoRepositoryInfo is the json type for a file that is written at the root
// of a 'proto_repository' rule (proto_repository_info.json).
type ProtoRepositoryInfo struct {
	VCS          string   `json:"vcs"`
	Commit       string   `json:"commit"`
	Tag          string   `json:"tag"`
	URLs         []string `json:"urls"`
	Sha256       string   `json:"sha256"`
	StripPrefix  string   `json:"strip_prefix"`
	SourceHost   string   `json:"source_host"`
	SourceOwner  string   `json:"source_owner"`
	SourceRepo   string   `json:"source_repo"`
	SourcePrefix string   `json:"source_prefix"`
	SourceCommit string   `json:"source_commit"`
}

// ProtoCompilerInfo is the json type for a file that is written at the by the
// 'proto_compiler' rule (proto_repository_info.json).
type ProtoCompilerInfo struct {
	Name        string `json:"name"`
	VersionFile string `json:"version_file"`
}
