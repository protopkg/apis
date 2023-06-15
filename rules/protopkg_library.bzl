def _protopkg_library_impl(ctx):
    info = ctx.attr.dep[ProtoInfo]
    srcs = [f.short_path for f in info.direct_sources]

    pkg = struct(
        name = ctx.label.name,
        package = ctx.label.package,
        srcs = srcs,
    )

    ctx.actions.write(ctx.outputs.pkg, pkg.to_json())
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
    },
    outputs = {
        "pkg": "%{name}.pkg",
    },
)
