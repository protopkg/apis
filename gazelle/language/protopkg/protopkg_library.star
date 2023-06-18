"""starlark rules definitions"""

def _make_protopkg_library_rule(rctx, pctx):
    r = gazelle.Rule(
        kind = "protopkg_library",
        name = pctx.proto_library.base_name + "_protopkg",
        attrs = {
            "proto": pctx.proto_library.name,
            "deps": [],
            "proto_compiler": "@//:proto_compiler",
            "proto_repository": "//:proto_repository",
            "visibility": rctx.visibility,
        },
    )
    return r

def _provide_protopkg_library(rctx, pctx):
    return struct(
        name = "protopkg_library",
        rule = lambda: _make_protopkg_library_rule(rctx, pctx),
        experimental_resolve_attr = "deps",
    )

protoc.Rule(
    name = "protopkg_library",
    load_info = lambda: gazelle.LoadInfo(name = "@com_github_protopkg_protoregistry//rules:protopkg_library.bzl", symbols = ["protopkg_library"]),
    kind_info = lambda: gazelle.KindInfo(resolve_attrs = {"deps": True}),
    provide_rule = _provide_protopkg_library,
)
