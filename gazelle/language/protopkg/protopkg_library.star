"""starlark rules definitions"""

def _make_protopkg_library_rule(rctx, pctx):
    r = gazelle.Rule(
        kind = "protopkg_library",
        name = pctx.proto_library.base_name + "_protopkg",
        attrs = {
            "dep": pctx.proto_library.name,
            "visibility": rctx.visibility,
        },
    )
    return r

def _provide_protopkg_library(rctx, pctx):
    return struct(
        name = "protopkg_library",
        rule = lambda: _make_protopkg_library_rule(rctx, pctx),
    )

protoc.Rule(
    name = "protopkg_library",
    load_info = lambda: gazelle.LoadInfo(name = "@protopkg_apis//rules:protopkg_library.bzl", symbols = ["protopkg_library"]),
    kind_info = lambda: gazelle.KindInfo(resolve_attrs = {"deps": True}),
    provide_rule = _provide_protopkg_library,
)
