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
