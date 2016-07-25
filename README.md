# fh-system-dump-tool

This repository contains the system dump tool for the RHMAP On-Prem product.

## Building

Building requires Go 1.6.

```
go build
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
