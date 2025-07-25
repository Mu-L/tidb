load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# Required by toolchains_protoc.
http_archive(
    name = "platforms",
    sha256 = "218efe8ee736d26a3572663b374a253c012b716d8af0c07e842e82f238a0a7ee",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/platforms/releases/download/0.0.10/platforms-0.0.10.tar.gz",
        "https://github.com/bazelbuild/platforms/releases/download/0.0.10/platforms-0.0.10.tar.gz",
    ],
)

http_archive(
    name = "bazel_features",
    sha256 = "ba1282c1aa1d1fffdcf994ab32131d7c7551a9bc960fbf05f42d55a1b930cbfb",
    strip_prefix = "bazel_features-1.15.0",
    url = "https://github.com/bazel-contrib/bazel_features/releases/download/v1.15.0/bazel_features-v1.15.0.tar.gz",
)

load("@bazel_features//:deps.bzl", "bazel_features_deps")

bazel_features_deps()

http_archive(
    name = "bazel_skylib",
    sha256 = "66ffd9315665bfaafc96b52278f57c7e2dd09f5ede279ea6d39b2be471e7e3aa",
    urls = [
        "http://bazel-cache.pingcap.net:8080/gomod/rules/bazel-skylib-1.4.2.tar.gz",
        "http://ats.apps.svc/gomod/rules/bazel-skylib-1.4.2.tar.gz",
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.4.2/bazel-skylib-1.4.2.tar.gz",
        "https://github.com/bazelbuild/bazel-skylib/releases/download/1.4.2/bazel-skylib-1.4.2.tar.gz",
    ],
)

load("@bazel_skylib//lib:versions.bzl", "versions")

versions.check(minimum_bazel_version = "6.0.0")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "9d72f7b8904128afb98d46bbef82ad7223ec9ff3718d419afb355fddd9f9484a",
    urls = [
        "http://bazel-cache.pingcap.net:8080/bazel-contrib/rules_go/releases/download/v0.55.1/rules_go-v0.55.1.zip",
        "http://ats.apps.svc/bazel-contrib/rules_go/releases/download/v0.55.1/rules_go-v0.55.1.zip",
        "https://cache.hawkingrei.com/bazel-contrib/rules_go/releases/download/v0.55.1/rules_go-v0.55.1.zip",
        "https://mirror.bazel.build/github.com/bazel-contrib/rules_go/releases/download/v0.55.1/rules_go-v0.55.1.zip",
        "https://github.com/bazel-contrib/rules_go/releases/download/v0.55.1/rules_go-v0.55.1.zip",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "7c40b746387cd0c9a4d5bb0b2035abd134b3f7511015710a5ee5e07591008dde",
    urls = [
        "http://bazel-cache.pingcap.net:8080/bazel-contrib/bazel-gazelle/releases/download/v0.43.0/bazel-gazelle-v0.43.0.tar.gz",
        "https://github.com/bazel-contrib/bazel-gazelle/releases/download/v0.43.0/bazel-gazelle-v0.43.0.tar.gz",
        "http://ats.apps.svc/bazel-contrib/bazel-gazelle/releases/download/v0.43.0/bazel-gazelle-v0.43.0.tar.gz",
        "https://cache.hawkingrei.com/bazel-contrib/bazel-gazelle/releases/download/v0.43.0/bazel-gazelle-v0.43.0.tar.gz",
    ],
)

http_archive(
    name = "rules_cc",
    sha256 = "d62624b45e0912713dcd3b8e30ba6ae55418ed6bf99e6d135cd61b8addae312b",
    strip_prefix = "rules_cc-0.1.2",
    urls = [
        "http://bazel-cache.pingcap.net:8080/bazelbuild/rules_cc/releases/download/0.1.2/rules_cc-0.1.2.tar.gz",
        "https://github.com/bazelbuild/rules_cc/releases/download/0.1.2/rules_cc-0.1.2.tar.gz",
        "http://ats.apps.svc/bazelbuild/rules_cc/releases/download/0.1.2/rules_cc-0.1.2.tar.gz",
    ],
)

http_archive(
    name = "rules_python",
    sha256 = "9f9f3b300a9264e4c77999312ce663be5dee9a56e361a1f6fe7ec60e1beef9a3",
    strip_prefix = "rules_python-1.4.1",
    urls = [
        "http://bazel-cache.pingcap.net:8080/bazel-contrib/rules_python/releases/download/1.4.1/rules_python-1.4.1.tar.gz",
        "https://github.com/bazel-contrib/rules_python/releases/download/1.4.1/rules_python-1.4.1.tar.gz",
        "http://ats.apps.svc/bazel-contrib/rules_python/releases/download/1.4.1/rules_python-1.4.1.tar.gz",
        "https://cache.hawkingrei.com/bazel-contrib/rules_python/releases/download/1.4.1/rules_python-1.4.1.tar.gz",
    ],
)

load("@rules_python//python:repositories.bzl", "py_repositories")

py_repositories()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")
load("@io_bazel_rules_go//go:deps.bzl", "go_download_sdk", "go_register_toolchains", "go_rules_dependencies")
load("//:DEPS.bzl", "go_deps")

# gazelle:repository_macro DEPS.bzl%go_deps
go_deps()

go_rules_dependencies()

go_download_sdk(
    name = "go_sdk",
    urls = [
        "https://cache.hawkingrei.com/golang/{}",
        "http://ats.apps.svc/golang/{}",
        "http://bazel-cache.pingcap.net:8080/golang/{}",
        "https://mirrors.aliyun.com/golang/{}",
        "https://dl.google.com/go/{}",
    ],
    version = "1.23.11",
)

gazelle_dependencies(go_sdk = "go_sdk")

go_register_toolchains(
    nogo = "@//build:tidb_nogo",
)

http_archive(
    name = "com_google_protobuf",
    integrity = "sha256-zl0At4RQoMpAC/NgrADA1ZnMIl8EnZhqJ+mk45bFqEo=",
    strip_prefix = "protobuf-29.0-rc2",
    # latest, as of 2021-03-08
    urls = [
        "https://github.com/protocolbuffers/protobuf/archive/v29.0-rc2.tar.gz",
        "https://mirror.bazel.build/github.com/protocolbuffers/protobuf/archive/v29.0-rc2.tar.gz",
    ],
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

http_archive(
    name = "remote_java_tools",
    sha256 = "f58a358ca694a41416a9b6a92b852935ad301d8882e5d22f4f11134f035317d5",
    urls = [
        "http://bazel-cache.pingcap.net:8080/gomod/rules/java_tools-v12.6.zip",
        "http://ats.apps.svc/gomod/rules/java_tools-v12.6.zip",
        "https://mirror.bazel.build/bazel_java_tools/releases/java/v12.6/java_tools-v12.6.zip",
        "https://github.com/bazelbuild/java_tools/releases/download/java_v12.6/java_tools-v12.6.zip",
    ],
)

http_archive(
    name = "remote_java_tools_linux",
    sha256 = "64294e91fe940c77e6d35818b4c3a1f07d78e33add01e330188d907032687066",
    urls = [
        "http://bazel-cache.pingcap.net:8080/gomod/rules/java_tools_linux-v12.6.zip",
        "http://ats.apps.svc/gomod/rules/java_tools_linux-v12.6.zip",
        "https://mirror.bazel.build/bazel_java_tools/releases/java/v12.6/java_tools_linux-v12.6.zip",
        "https://github.com/bazelbuild/java_tools/releases/download/java_v12.6/java_tools_linux-v12.6.zip",
    ],
)

http_archive(
    name = "rules_proto",
    sha256 = "303e86e722a520f6f326a50b41cfc16b98fe6d1955ce46642a5b7a67c11c0f5d",
    strip_prefix = "rules_proto-6.0.0",
    urls = [
        "https://github.com/bazelbuild/rules_proto/releases/download/6.0.0/rules_proto-6.0.0.tar.gz",
    ],
)

load("@rules_proto//proto:repositories.bzl", "rules_proto_dependencies")

rules_proto_dependencies()

load("@rules_proto//proto:toolchains.bzl", "rules_proto_toolchains")

rules_proto_toolchains()

http_archive(
    name = "rules_java",
    sha256 = "f5a3e477e579231fca27bf202bb0e8fbe4fc6339d63b38ccb87c2760b533d1c3",
    strip_prefix = "rules_java-981f06c3d2bd10225e85209904090eb7b5fb26bd",
    urls = [
        "http://bazel-cache.pingcap.net:8080/gomod/rules/rules_java/rules_java-981f06c3d2bd10225e85209904090eb7b5fb26bd.tar.gz",
        "http://ats.apps.svc/bazelbuild/gomod/rules/rules_java/rules_java-981f06c3d2bd10225e85209904090eb7b5fb26bd.tar.gz",
        "https://mirror.bazel.build/github.com/bazelbuild/rules_java/archive/981f06c3d2bd10225e85209904090eb7b5fb26bd.tar.gz",
        "https://github.com/bazelbuild/rules_java/archive/981f06c3d2bd10225e85209904090eb7b5fb26bd.tar.gz",
    ],
)

http_archive(
    name = "toolchains_protoc",
    sha256 = "117af61ee2f1b9b014dcac7c9146f374875551abb8a30e51d1b3c5946d25b142",
    strip_prefix = "toolchains_protoc-0.3.0",
    url = "https://github.com/aspect-build/toolchains_protoc/releases/download/v0.3.0/toolchains_protoc-v0.3.0.tar.gz",
)
