def _protopkg_library_impl(ctx):
    info = ctx.attr.dep[ProtoInfo]

    # srcs = [f.short_path for f in info.direct_sources]
    protoset = info.direct_descriptor_set

    args = ctx.actions.args()
    args.add("-repo", ctx.label.workspace_name)
    args.add("-pkg", ctx.label.package)
    args.add("-name", ctx.label.name)
    args.add("-protoset", protoset.path)
    args.add("-output", ctx.outputs.pkg)

    ctx.actions.run(
        executable = ctx.executable._tool,
        arguments = [args],
        inputs = [protoset],
        outputs = [ctx.outputs.pkg],
    )

    return [DefaultInfo(
        files = depset([ctx.outputs.pkg]),
    )]

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
        "pkg": "%{name}.pkg",
    },
)
