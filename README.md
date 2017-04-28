# fh-system-dump-tool

This repository contains the system dump tool for the Red Hat Mobile Application
Platform, RHMAP.

It is used to aggregate debugging information for RHMAP installations running on
top of OpenShift.


## Runtime requirements

The dump tool depends on some commands being available:

- OpenShift command line interface, `oc`
  ([installation instructions](https://docs.openshift.com/enterprise/3.2/cli_reference/get_started_cli.html#installing-the-cli))

  Used to fetch data from an existing OpenShift cluster.

- GNU tar (optional)

  Used to generate a `.tar.gz` archive with the dump data.


## Running

1. Login to an OpenShift cluster where RHMAP is installed

    Login as an user with sufficient permissions to list resources in the
    RHMAP-related projects. Ideally, the user has the `cluster-reader` role,
    what gives it read access to cluster level resources such as persistent
    volumes and nodes.

    ```
    oc login <public-master-url>
    ```

2. Run the system dump tool:

    ```
    fh-system-dump-tool
    ```

3. The above command will print where the archive with debugging information was
stored, and any potential problem detected.


## Developing and Contributing

See [the contribution guide](CONTRIBUTING.md).


## License

See [LICENSE](LICENSE) file.

## Understanding the output

Once the system dump tool has completed running it will create a `.tar.gz` file which contains the results of the analysis. This should be extracted into a directory as the data in this archive is intended to be human-readable.

The first file that should be consulted to identify potential issues is the `analysis.json` file in the root of the dump directory. This file can quickly point out common problems that can occur in RHMAP products.

### Analysis.json
This file is formatted as JSON and has a projects block in which each project's various analytical tests results are written. If one or more of these tests are failing against an RHMAP project; it is very likely to be identifying issues and deserves further investigation.

At the time of writing, the dump tool runs the follow tests:
- check number of replicas in deployment configs
- check pods for containers in waiting state
- check event log for errors

#### Check number of replicas in deployment configs
This test will check that no deployment configs have invalid replica values, for example 0, which could cause issues for the given project.

#### Check Pods for Containers in waiting state
A waiting container has failed to launch yet for some reason, this is usually combined with errors in the eventlog which may help explain why the container is failing to launch (e.g. cannot schedule the container).

#### Check Event Log For Errors
This test is looking in the event log for any errors that occurred within the given project. If any are found they are logged here, this is the most likely test to give false positives; nevertheless any errors in the event log are worth reading, and keeping in mind when investigating other issues.

### oc_adm_diagnostics
This is a good next stop if the analysis.json file did not shed light on the issue being investigated. This file contains the raw output of executing `oc adm diagnostics` against the Openshift Cluster. This command runs the Openshift Diagnostics tool against the underlying Openshift cluster and logs any potential issues with this layer of the solution.

### Projects Directory
At this point, some errors should have been discovered, but more information may be required to fully understand why they are happening. The projects directory is a good resource to turn to at this point as it contains extremely verbose details about every resource in every project. Here is a list of where each piece of information can be found:

Resource | Location | Notes 
--- | --- | ---
Pod Logs | `project/<project-name>/logs` | 
previous pod logs | `project/<project-name>/logs-previous` |
Nagios current status | `project/<project-name>/nagios/<nagios-pod>_status.dat` | This resembles JSON but is in fact a bespoke Nagios format
Nagios historical data | `project/<project-name>/nagios/<nagios-pod>_history.tar` | This will need to be unarchived
Other resources | `project/<project-name>/definitions/<resource>/json` | Definition of resources such as configmaps, deploymentconfigs, etc
 
### Meta directory
If the archive appears to be missing a lot of critical data, or contains a lot of errors suggesting it cannot access required resources, the meta directory can be useful in finding out what the logged in user did and did not have permission to acces. although the system dump tool will always make a best effort to provide some sort of information; insufficient access to the cluster could render the output almost entirely unreliable.

### Other notes
#### Handling of errors during dump procedure
When a command is executed to retrieve information from the cluster / project it's output is stored in a file named after the command executed; this file will be created whether or not the command worked. However if there is any output on `STDERR` during the operation a new file will be created with the same name and `.stderr` appended to it. If this file exists it should be consulted first to ascertain whether the actual output file is reliable. 
