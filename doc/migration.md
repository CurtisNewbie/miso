# Upgrade & Migration

miso may introduce breaking changes. To migrate to latest release, you can use gopatch to (hopefully) automatically migrate to latest release.

Install [uber-go/gopatch](https://github.com/uber-go/gopatch).

```sh
go install github.com/uber-go/gopatch@latest
```

Upgrade your project to latest release:

```sh
go get github.com/curtisnewbie/miso@v0.3.0
```

Run the following command to apply patch (v0.3.0.patch is just an example),

e.g.,

```sh
gopatch -p "$miso_home/patch/v0.3.0.patch" ./...
```

Use git diff to see the changes and run your project to see if it still works.

# misopatch

You can also install `misopatch` tool (see [Tools](./tools.md)). It embeds the patch files in binary, installs gopatch if missing, and applies all the patches for you.

Always install the latest misopatch version using tags (do not use `go get -u`), e.g.,

```sh
go install github.com/curtisnewbie/miso/cmd/misopatch@v0.3.6
```

And then run misopatch:

```sh
$ misopatch
# gopatch not found, installing
# Found 3 patches
# Applied patch: v0.3.0.patch
# Applied patch: v0.3.5.patch
# Applied patch: v0.3.6.patch
```