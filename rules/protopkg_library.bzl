load("//rules:providers.bzl", "ProtoCompilerInfo", "ProtoPackageInfo")

def _protopkg_library_impl(ctx):
    proto_repository_info_files = ctx.attr.proto_repository_info[DefaultInfo].files.to_list()
    if len(proto_repository_info_files) != 1:
        fail("expected a single file for in the label list for 'proto_repository_info'")

    proto_info = ctx.attr.proto[ProtoInfo]
    proto_descriptor_set_file = proto_info.direct_descriptor_set
    proto_compiler_info = ctx.attr.proto_compiler[ProtoCompilerInfo]
    proto_repository_info_file = proto_repository_info_files[0]
    proto_compiler_info_file = ctx.actions.declare_file(ctx.label.name + ".compiler.info.json")
    proto_compiler_version_file = proto_compiler_info.version_file

    ctx.actions.write(proto_compiler_info_file, struct(
        name = proto_compiler_info.name,
        version_file = proto_compiler_version_file.path,
    ).to_json())

    args = ctx.actions.args()
    args.add("-proto_descriptor_set_file", proto_descriptor_set_file.path)
    args.add("-proto_repository_info_file", proto_repository_info_file.path)
    args.add("-proto_compiler_info_file", proto_compiler_info_file.path)
    args.add("-proto_compiler_version_file", proto_compiler_version_file.path)

    inputs = [
        proto_descriptor_set_file,
        proto_compiler_info_file,
        proto_compiler_version_file,
        proto_repository_info_file,
    ]

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
        ProtoPackageInfo(
            label = ctx.label,
            proto_package_file = ctx.outputs.proto,
            proto_info = proto_info,
        ),
    ]

protopkg_library = rule(
    implementation = _protopkg_library_impl,
    attrs = {
        "proto": attr.label(
            doc = "proto_library dependency",
            mandatory = True,
            providers = [ProtoInfo],
        ),
        "deps": attr.label_list(
            doc = "protopkg_library dependencies",
            providers = [ProtoPackageInfo],
        ),
        "proto_repository_info": attr.label(
            mandatory = True,
            allow_files = True,
        ),
        "proto_compiler": attr.label(
            mandatory = True,
            providers = [ProtoCompilerInfo],
        ),
        "_tool": attr.label(
            default = str(Label("//cmd/protopkg_library")),
            executable = True,
            cfg = "exec",
        ),
    },
    outputs = {
        "proto": "%{name}.pkg.pb",
        "json": "%{name}.pkg.json",
    },
)
