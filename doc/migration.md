# Upgrade & Migration

miso (I) may introduce breaking changes. To migrate to latest release, you can use gopatch to (hopefully) automatically migrate to latest release.

Install [uber-go/gopatch](https://github.com/uber-go/gopatch).

```sh
go install github.com/uber-go/gopatch@latest
```

Go to your project directory, run the following command to apply patch:

```sh
miso_patch_path="$miso_home/patch/v0.3.0.path"

gopatch -p "$miso_patch_path" ./...
```

Then use git diff to see the changes.