load("//rules:providers.bzl", "ProtoPackageInfo")

def _protopkg_library_impl(ctx):
    proto_info = ctx.attr.dep[ProtoInfo]

    protoset = proto_info.direct_descriptor_set

    args = ctx.actions.args()
    args.add("-repo", ctx.label.workspace_name)
    args.add("-pkg", ctx.label.package)
    args.add("-name", ctx.label.name)
    args.add("-protoset", protoset.path)

    ctx.actions.run(
        executable = ctx.executable._tool,
        arguments = [args] + ["-proto_out", ctx.outputs.proto.path],
        inputs = [protoset],
        outputs = [ctx.outputs.proto],
    )
    ctx.actions.run(
        executable = ctx.executable._tool,
        arguments = [args] + ["-json_out", ctx.outputs.json.path],
        inputs = [protoset],
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
            proto_package_file = ctx.outputs.proto,
            proto_info = proto_info,
        ),
    ]

protopkg_library = rule(
    implementation = _protopkg_library_impl,
    attrs = {
        "dep": attr.label(
            doc = "proto_library dependency",
            mandatory = True,
            providers = [ProtoInfo],
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
