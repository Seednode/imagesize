## About
Sometimes, you just need a way to view all images matching specific dimension requirements on your machine.

Simply point this tool at one or more directories and specify what you want to display (images over 512 pixels wide? under 256 pixels high?).

For example, to view all images wider than 512 pixels in a directory, you might want to run `imagesize width over 512 -r ~/path/here`.

You will be presented with a sorted list of all matching files (by default, sorted by name in ascending order) in that directory and any of its children.

You can also pass the `-v|--verbose` flag to have the dimensions appended to the output for each image.

Feature requests, code criticism, bug reports, general chit-chat, and unrelated angst accepted at `imagesize@seedno.de`.

Static binary builds available [here](https://cdn.seedno.de/builds/imagesize).

x86_64 and ARM Docker images of latest version: `oci.seedno.de/seednode/imagesize:latest`.

Dockerfile available [here](https://git.seedno.de/seednode/imagesize/raw/branch/master/docker/Dockerfile).

## Usage output
```
displays images matching the specified constraints

Usage:
  imagesize [command]

Available Commands:
  height      Filter images by height
  width       Filter images by width

Flags:
  -h, --help                  help for imagesize
  -c, --max-concurrency int   maximum number of paths to scan at once (default 4096)
  -e, --or-equal              also match files equal to the specified dimension
  -r, --recursive             include subdirectories
  -k, --sort-key string       sort output by the specified key (height, width, name) (default "name")
  -o, --sort-order string     sort output in the specified direction (asc[ending], desc[ending]) (default "ascending")
  -v, --verbose               display image dimensions and total matched file count
  -V, --version               display version and exit
```

## Building the Docker image
From inside the cloned repository, build the image using the following command:

`REGISTRY=<registry url> LATEST=yes TAG=alpine ./build-docker.sh`
