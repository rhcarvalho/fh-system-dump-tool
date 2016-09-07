# fh-system-dump-tool

This repository contains the system dump tool for the RHMAP On-Prem product.

## Building

Building requires Go 1.6 or later. The code can be built using the standard Go
tools, `go build`, `go install` and `go get`. However, use `make` for release
binaries that include version information:

```
make
```

## Runtime Prerequisites

- Installation of [openshift-cli](https://docs.openshift.com/enterprise/3.2/cli_reference) for `oc` binary.

## Running

The follow section outlines the steps required to run the system dump tool.

### 1. Login to OpenShift Cluster as an Administrative User

```
oc login <public-master-url>
```

### 2. Run the System Dump Tool

```
./fh-system-dump-tool
```

## Adding new analysis checks
Create a function - currently all in analysis.go - which matches the CheckTask interface:
```
type CheckTask func(string, io.Writer) (Result, error)
```

The writer is where the stderr output from your checks should be sent.

If a resource from oc is required, you can use the helper function: `getResourceStruct` pass to this the current 
project, the resource type and a pointer to the struct the json should decode into.

The Result struct has the following properties:
- CheckName
- Status
- StatusMessage
- Info (Array)
  - Name
  - Namespace
  - Kind
  - Count
  - Message

Update the function `CheckTasks` to also return your new check function.

## Releasing

* Tag a new version, e.g., `v0.1.0`
* Create a new __Release__ from the [releases](https://github.com/feedhenry/fh-system-dump-tool/releases) page
* Add some info about the release
* Build a release binary using `make`
* Upload the built binary
* Publish it
