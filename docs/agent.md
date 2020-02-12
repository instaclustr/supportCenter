# Agent
The Instaclustr collection agent is a command line tool that makes collecting information about your Cassandra cluster easier. It will SSH into a list of Cassandra nodes and collect the output of a number of nodetool commands, logs and configuration files. It will then compress all evidence into a single tarball ready for submission to Instaclustr as part of a support ticket. This will help dramatically speed up issue resolution.

## Quickstart
To agent supports the following command line flags:

* `-disable_known_hosts` - Skip loading the user’s known-hosts file
* `-l USER` - User to log in as on the remote machine
* `-mc HOST/IP` - Metrics collecting hostname. E.g. the prometheus server.
* `-nc HOST/IP` - Node collecting hostnames - This can be a comma seperated list of nodes
* `-p int` - Port to connect to on the remote host (default 22) via SSH
* `-pk PATH` - List of files from which the identification keys (private key) for public key authentication are read

E.g. `./agent -disable_known_hosts -l ubuntu -mc 10.0.56.1 -nc 10.0.0.1,10.0.0.2,10.0.0.3,10.0.0.4 -pk ~/.ssh/id_rsa`

The agent will then collect data from the nodes and prometheus server and store the resulting tarball (and intermediate results) in a folder call `data/` (from your working directory).

The agent also supports a settings file which allows you to control the expected location for various log and configuration files. Currently the agent will look for settings.yml in the working directory which you launch the tool from. In the future, you can define the settings file via a command line flag and the default location will be in a "dot" folder e.g `~/.instaclustr`.

An example settings.yml file is below:

```yaml
# Node collection
node:
  cassandra:
    config-path: "/etc/cassandra"
    log-path: "/var/log/cassandra"
    home-path: "/var/lib/cassandra"
    data-path:
      - "/var/lib/cassandra/data"
  collecting:
    configs:
      - "cassandra.yaml"
      - "cassandra-env.sh"
      - "jvm.options"
      - "logback.xml"
    logs:
      - "system.log"

# Metrics collecting
metrics:
  prometheus:
    port: 9090
    data-path: "/prometheus/data/"
```
