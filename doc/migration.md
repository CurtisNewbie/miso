# Upgrade & Migration

miso (I) may introduce breaking changes. To migrate to latest release, you can use gopatch to (hopefully) automatically migrate to latest release.

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
miso_patch_path="$miso_home/patch/v0.3.0.patch"

gopatch -p "$miso_patch_path" ./...
```

Use git diff to see the changes and run your project to see if it still works.