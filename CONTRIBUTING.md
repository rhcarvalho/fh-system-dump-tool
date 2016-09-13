# Contributing

This is a brief guide for contributors.


## Building

Building requires [Go 1.6 or later](https://golang.org/doc/install). Clone this
repository to your [Go workspace](https://golang.org/doc/code.html#Workspaces)
defined via [the GOPATH environment
variable](https://golang.org/doc/code.html#GOPATH).

The code can be built using the standard Go tools, `go build`, `go install` and
`go get`. If you haven't cloned the repository already, a quick way to get it
into your GOPATH is:

```
go get github.com/feedhenry/fh-system-dump-tool
```

However, use `make` for release binaries that include version information:

```
make
```

The command above will install the `fh-system-dump-tool` to your GOPATH/bin
directory. It is recommended that GOPATH/bin is part of your PATH environment
variable.


## Testing

Pull Requests are automatically tested using [Travis
CI](https://travis-ci.org/). You can run the same tests locally during development:

```
make -k ci
```

That will run several verifications involving code formatting, unit tests, etc.
The `-k` flag tells `make` to keep running even if one of the verifications
fail. If you want to to terminate after encountering the first problem, omit
that flag.

To see what commands are used for testing, look at the output of `make -n ci`.
Refer to the [Makefile](Makefile) to see how each verification is implemented.


## Releasing

To release a new version:

* Tag a new version, e.g., `v0.1.0`
* Create a new __Release__ from the [releases](https://github.com/feedhenry/fh-system-dump-tool/releases) page
* Add some info about the release
* Build a release binary using `make`
* Upload the built binary
* Publish it
