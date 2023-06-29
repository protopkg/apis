"""providers.bzl: public facing bazel providers
"""

ProtoPackageInfo = provider(
    "info about a protopkg_package rule",
    fields = {
        "label": "the label of the protopkg_file rule",
        "output_file": "the generated proto-encoded ProtoPackage file (type https://bazel.build/rules/lib/builtins/File)",
        "deps": "the direct ProtoFileInfo direct dependencies of this one",
    },
)

ProtoFileInfo = provider(
    "info about a protopkg_file rule",
    fields = {
        "label": "the label of the protopkg_file rule",
        "output_file": "the generated proto-encoded ProtoPackage file (type https://bazel.build/rules/lib/builtins/File)",
        "proto_file_direct_deps": "the direct ProtoFileInfo direct dependencies of this one",
        "proto_info": "the underlying ProtoInfo provider (type https://docs.bazel.build/versions/5.4.1/skylark/lib/ProtoInfo.html)",
    },
)

ProtoCompilerInfo = provider(
    "info about a protobuf compiler",
    fields = {
        "name": "the name of the compiler",
        "version_file": "the file that contains the version infomation",
    },
)
