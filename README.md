# fh-system-dump-tool
This repository contains the system dump tool for the RHMAP On-Prem product

##Prerequisites
Installation of openshift-cli for oc binary
https://docs.openshift.com/enterprise/3.2/cli_reference

##Running
The follow section outlines the steps required to run the system dump tool

####Login to Openshift Cluster as an Administrative User
```bash
oc login <public-master-url>
```
####Run System Dump Tool
```bash
./dump
```

##View Results
Once the system dump tool has been run, a new archived report is generated in the reports directory of this repository. See example below

```bash
ls ./reports/
report_2016-07-12_100544.tar.gz
```
