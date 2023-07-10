load("@build_stack_rules_proto//rules:providers.bzl", "ProtoRepositoryInfo")
load("//rules:providers.bzl", "ProtoCompilerInfo", "ProtoFileInfo")

def _protopkg_file_impl(ctx):
    proto_repository_info = ctx.attr.proto_repository[ProtoRepositoryInfo]
    proto_compiler_info = ctx.attr.proto_compiler[ProtoCompilerInfo]
    proto_info = ctx.attr.proto[ProtoInfo]
    proto_descriptor_set_file = proto_info.direct_descriptor_set
    proto_compiler_version_file = proto_compiler_info.version_file

    direct_deps = [dep[ProtoFileInfo] for dep in ctx.attr.deps]
    direct_deps_files = depset([info.output_file for info in direct_deps])
    direct_source_files = proto_info.direct_sources
    transitive_deps = [dep[ProtoFileInfo].proto_file_transitive_depset for dep in ctx.attr.deps]

    args = ctx.actions.args()
    args.add("-proto_descriptor_set_file", proto_descriptor_set_file.path)
    args.add("-proto_repository_host", proto_repository_info.source_host)
    args.add("-proto_repository_owner", proto_repository_info.source_owner)
    args.add("-proto_repository_repo", proto_repository_info.source_repo)
    args.add("-proto_repository_commit", proto_repository_info.source_commit)
    args.add("-proto_repository_root", proto_repository_info.source_prefix)
    args.add("-proto_compiler_name", proto_compiler_info.name)
    args.add("-proto_compiler_version_file", proto_compiler_version_file.path)
    args.add_joined(
        "-proto_file_direct_dependency_files",
        [f.path for f in direct_deps_files.to_list()],
        join_with = ",",
    )
    args.add_joined(
        "-proto_source_files",
        [f.path for f in direct_source_files],
        join_with = ",",
    )

    # print(" args:", args)

    inputs = [
        proto_descriptor_set_file,
        proto_compiler_version_file,
    ] + direct_source_files + direct_deps_files.to_list()

    ctx.actions.run(
        executable = ctx.executable._tool,
        arguments = [args] + ["-proto_out", ctx.outputs.proto.path],
        inputs = inputs,
        outputs = [ctx.outputs.proto],
    )

    ctx.actions.run(
        executable = ctx.executable._tool,
        arguments = [args] + ["-json_out", ctx.outputs.json.path],
        inputs = inputs,
        outputs = [ctx.outputs.json],
    )

    return [
        DefaultInfo(
            files = depset([ctx.outputs.proto]),
        ),
        OutputGroupInfo(
            json = depset([ctx.outputs.json]),
        ),
        ProtoFileInfo(
            label = ctx.label,
            output_file = ctx.outputs.proto,
            proto_file_direct_deps = direct_deps,
            proto_file_transitive_depset = depset(direct_deps, transitive = transitive_deps),
            proto_info = proto_info,
        ),
    ]

_protopkg_file = rule(
    implementation = _protopkg_file_impl,
    attrs = {
        "proto": attr.label(
            doc = "proto_library dependency",
            mandatory = True,
            providers = [ProtoInfo],
        ),
        "deps": attr.label_list(
            doc = "protopkg_file dependencies",
            providers = [ProtoFileInfo],
        ),
        "proto_repository": attr.label(
            mandatory = True,
            providers = [ProtoRepositoryInfo],
        ),
        "proto_compiler": attr.label(
            mandatory = True,
            providers = [ProtoCompilerInfo],
        ),
        "_tool": attr.label(
            default = str(Label("//cmd/protopkg_file")),
            executable = True,
            cfg = "exec",
        ),
    },
    outputs = {
        "proto": "%{name}.pkg.pb",
        "json": "%{name}.pkg.json",
    },
)

def protopkg_file(**kwargs):
    name = kwargs.pop("name")

    _protopkg_file(name = name, **kwargs)
