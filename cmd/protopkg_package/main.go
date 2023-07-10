package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"

	pppb "github.com/stackb/apis/build/stack/protobuf/package/v1alpha2"
	"github.com/stackb/protoreflecthash"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type flagName string

type config struct {
	DirectDeps     []string `json:"direct_deps"`
	TransitiveDeps []string `json:"transitive_deps"`
}

type protoPackageFile struct {
	pkg  *pppb.ProtoPackage
	file *pppb.ProtoFile
}

const (
	configFileJsonFlagName  flagName = "config_json_file"
	protoOutputFileFlagName flagName = "proto_out"
	jsonOutputFileFlagName  flagName = "json_out"
)

var (
	configJsonFile  = flag.String(string(configFileJsonFlagName), "", "path to json config file (containing string array of deps file names)")
	protoOutputFile = flag.String(string(protoOutputFileFlagName), "", "path of file to write the generated proto file")
	jsonOutputFile  = flag.String(string(jsonOutputFileFlagName), "", "path of file to write the generated json file")
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
	}
}

func run() error {
	flag.Parse()

	cfg, err := readConfigJsonFile(configFileJsonFlagName, *configJsonFile)
	if err != nil {
		return err
	}

	var directPkgs []*pppb.ProtoPackage
	for _, filename := range cfg.DirectDeps {
		fileDep, err := readProtoPackageFile(configFileJsonFlagName, filename)
		if err != nil {
			return err
		}
		directPkgs = append(directPkgs, fileDep)
	}
	var transitivePkgs []*pppb.ProtoPackage
	for _, filename := range cfg.TransitiveDeps {
		fileDep, err := readProtoPackageFile(configFileJsonFlagName, filename)
		if err != nil {
			return err
		}
		transitivePkgs = append(transitivePkgs, fileDep)
	}

	pkgset, err := makeProtoPackageSet(directPkgs, transitivePkgs)
	if err != nil {
		return err
	}

	if *protoOutputFile != "" {
		if err := writeProtoOutputFile(pkgset, *protoOutputFile); err != nil {
			return err
		}
	}
	if *jsonOutputFile != "" {
		if err := writeJsonOutputFile(pkgset, *jsonOutputFile); err != nil {
			return err
		}
	}

	return nil
}

func readConfigJsonFile(flag flagName, filename string) (*config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func readProtoPackageFile(flag flagName, filename string) (*pppb.ProtoPackage, error) {
	if filename == "" {
		return nil, errorFlagRequired(flag)
	}

	var pp pppb.ProtoPackage
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", flag, err)
	}
	if err := proto.Unmarshal(data, &pp); err != nil {
		return nil, fmt.Errorf("unmarshaling %s: %w", flag, err)
	}
	return &pp, nil
}

func writeProtoOutputFile(msg proto.Message, filename string) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling generated data: %v", err)
	}
	if err := os.WriteFile(filename, data, os.ModePerm); err != nil {
		return fmt.Errorf("writing proto file: %w", err)
	}
	return nil
}

func writeJsonOutputFile(msg proto.Message, filename string) error {
	marshaler := protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}
	jsonstr, err := marshaler.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling json: %w", err)
	}
	if err := os.WriteFile(filename, []byte(jsonstr), os.ModePerm); err != nil {
		return fmt.Errorf("writing json file: %w", err)
	}
	return nil
}

func makeProtoPackageSet(directPkgs, transitivePkgs []*pppb.ProtoPackage) (*pppb.ProtoPackageSet, error) {
	byPackageName := make(map[string][]*protoPackageFile)
	for _, pkg := range directPkgs {
		for _, file := range pkg.Files {
			name := *file.File.Package
			byPackageName[name] = append(byPackageName[name], &protoPackageFile{
				file: file,
				pkg:  pkg,
			})
		}
	}
	for _, pkg := range transitivePkgs {
		for _, file := range pkg.Files {
			name := *file.File.Package
			byPackageName[name] = append(byPackageName[name], &protoPackageFile{
				file: file,
				pkg:  pkg,
			})
		}
	}

	packageNames := make([]string, 0, len(byPackageName))
	for packageName := range byPackageName {
		packageNames = append(packageNames, packageName)
	}
	sort.Strings(packageNames)

	var pkgset pppb.ProtoPackageSet
	for _, packageName := range packageNames {
		pkgFiles := byPackageName[packageName]
		files := make([]*pppb.ProtoFile, len(pkgFiles))
		rep := pkgFiles[0]
		for i, pkgFile := range pkgFiles {
			files[i] = pkgFile.file
		}
		pkg, err := makeProtoPackage(rep.pkg.Archive, rep.pkg.Compiler, packageName, files)
		if err != nil {
			return nil, err
		}
		pkgset.Packages = append(pkgset.Packages, pkg)
	}

	providesFile := make(map[string]*pppb.ProtoPackage)
	for _, dep := range pkgset.Packages {
		for _, file := range dep.Files {
			providesFile[*file.File.Name] = dep
		}
	}
	for _, pkg := range pkgset.Packages {
		for _, file := range pkg.Files {
			for _, dep := range file.File.Dependency {
				provider, ok := providesFile[dep]
				if !ok {
					log.Fatalln("unknown provider for:", dep)
				}
				log.Println(pkg.Name, dep, "provider:", provider.Name)

				if provider == pkg {
					continue
				}
				pkg.Dependencies = append(pkg.Dependencies, makeProtoPackageDependency(provider))
				log.Println(pkg.Name, "deps:", pkg.Dependencies)
			}
		}
	}

	return &pkgset, nil
}

func makeProtoPackage(archive *pppb.ProtoArchive, compiler *pppb.ProtoCompiler, name string, files []*pppb.ProtoFile) (*pppb.ProtoPackage, error) {
	sort.Slice(files, func(i, j int) bool {
		a := files[i]
		b := files[j]
		return *a.File.Name < *b.File.Name
	})

	hash, err := makeProtoPackageHash(files)
	if err != nil {
		return nil, fmt.Errorf("calculating proto package hash: %w", err)
	}

	return &pppb.ProtoPackage{
		Name:     name,
		Archive:  archive,
		Compiler: compiler,
		Files:    files,
		Hash:     hash,
	}, nil
}

func makeProtoPackageDependency(pkg *pppb.ProtoPackage) string {
	root := "~"
	if pkg.Archive.Root != "" {
		root = pkg.Archive.Root
	}
	return fmt.Sprintf("%s/%s/%s:%s", pkg.Archive.Repository.FullName, pkg.Archive.ShortSha1, root, pkg.Name)
}

func errorFlagRequired(name flagName) error {
	return fmt.Errorf("flag required but not provided: -%s", name)
}

func protoreflectHash(msg proto.Message) (string, error) {
	hasher := protoreflecthash.NewHasher()
	data, err := hasher.HashProto(msg.ProtoReflect())
	if err != nil {
		return "", fmt.Errorf("hashing proto: %w", err)
	}
	return fmt.Sprintf("protoreflecthash.v0:%x", data), nil
}

func makeProtoPackageHash(files []*pppb.ProtoFile) (string, error) {
	stripped := make([]*pppb.ProtoFile, len(files))
	for i, file := range files {
		strip := proto.Clone(file).(*pppb.ProtoFile)
		strip.SourceCode = ""
		strip.FileSha256 = ""
		strip.FileSize = 0
		strip.File.SourceCodeInfo = nil
		stripped[i] = strip
	}
	return protoreflectHash(&pppb.ProtoPackage{
		Files: stripped,
	})
}
