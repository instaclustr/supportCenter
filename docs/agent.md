# Agent
The Instaclustr collection agent is a command line tool that makes collecting information about your Cassandra cluster easier. It will SSH into a list of Cassandra nodes and collect the output of a number of nodetool commands, logs and configuration files. It will then compress all evidence into a single tarball ready for submission to Instaclustr as part of a support ticket. This will help dramatically speed up issue resolution.

## Quickstart
To agent supports the following command line flags:

* `-disable_known_hosts` - Skip loading the user’s known-hosts file
* `-l USER` - User to log in as on the remote machine
* `-mc HOST/IP` - Metrics collecting hostname. E.g. the prometheus server.
* `-mc-from "DATETIME"` - Datetime (RFC3339 format, 2006-01-02T15:04:05Z07:00) to fetch metrics from some time point. (Default 1970-01-01 00:00:00 +0000 UTC)
* `-mc-to "DATETIME"` - Datetime (RFC3339 format, 2006-01-02T15:04:05Z07:00) to fetch metrics to some time point. (Default current datetime)
* `-nc HOST/IP` - Node collecting hostnames - This can be a comma separated list of nodes
* `-p int` - Port to connect to on the remote host (default 22) via SSH
* `-pk PATH` - List of files from which the identification keys (private key) for public key authentication are read, in addition to default one (Default [HOME]/.ssh/id_rsa)
* `-config PATH` - The path to the configuration file
* `generate-config PATH` - The path where the default settings file will be created

E.g. `./agent -disable_known_hosts -l ubuntu -mc 10.0.56.1 -nc 10.0.0.1,10.0.0.2,10.0.0.3,10.0.0.4 -pk ~/.ssh/id_rsa`

**Examples:**

_Fetch metrics by specific time span_
```shell script
./agent -disable_known_hosts -l ubuntu -nc 10.0.0.1,10.0.0.2 -mc metrics.example.com -mc-from "2020-02-18T00:00:00Z" -mc-to "2020-02-20T00:00:00Z"
```


The agent will then collect data from the nodes and prometheus server and store the resulting tarball (and intermediate results) in a data folder (the path can be configured in the settings `agent.collected-data-path`, default path `~/.instaclustr/supportcenter/DATA`).

The agent also supports a settings file which allows you to control the expected location for various log and 
configuration files.  
Configuration file search order:
* Defined via a command line flag `-config`
* The dotfiles directory `~/.instaclustr/supportcenter` that can contain multiple config files. By the default profile `~/.instaclustr/supportcenter/DEFAULT` (which has the name of the config file as its contents).
* Default one, `settings.yaml` in the working dir

The agent will collect data from the nodes specified in the settings (`target.nodes`, `target.metrics`) and in the command line arguments (`-nc`, `-mc`).

_Example (settings file)_
```yaml
# Common settings
agent:
  collected-data-path: "~/.instaclustr/supportcenter/DATA"

# Collecting settings
node:
  cassandra:
    config-path: "/etc/cassandra"
    log-path: "/var/log/cassandra"
    gc-path:  "/var/log/cassandra"
    data-path:
      - "/var/lib/cassandra/data"
    username: "JMX username"
    password: "JMX password"
  collecting:
    configs:
      - "cassandra.yaml"
      - "cassandra-env.sh"
      - "jvm.options"
      - "logback.xml"
    logs:
      - "system.log"
    gc-log-patterns:
      - "gc*"
    info:
      command-wrapper:
        common: "sudo"
        nodetool: ""
metrics:
  prometheus:
    port: 9090
    data-path: "/prometheus/data/"

# Collecting targets (node and metric hostnames)
target:
  nodes:
    - '10.0.0.1'
    - '10.0.0.2'
  metrics:
    - 'metrics.example.com'
```

### Settings
* **node.cassandra.config-path** - path for cassandra configuration files
* **node.collecting.configs** - list of configuration files to be collected
* **node.cassandra.log-path** - path for cassandra log files
* **node.collecting.logs** - list of log files to be collected
* **node.cassandra.gc-path** - path for cassandra garbage collector log files
* **node.collecting.gc-log-patterns** - list of patterns that will be used to select files from the garbage collector directory (See [Pattern](https://golang.org/pkg/path/filepath/#Match))
* **node.cassandra.data-path** - List of directories where the DiscInfo test will be performed
* **node.cassandra.username** - Remote JMX agent username (used by nodetool)
* **node.cassandra.password** - Remote JMX agent password (used by nodetool)
* **node.collecting.info.command-wrapper.common** - Command wrapper (prefix), which are called when collecting information about the environment (except nodetool)
* **node.collecting.info.command-wrapper.nodetool** - Command wrapper (prefix), which are called when collecting information about the environment (**nodetool** only)

## Cassandra deployment requirements
This collection agent depends on having a properly configured and running Prometheus metrics server running and collecting metrics from your Cassandra cluster in combination with the cassandra-exporter. For instructions on setting up cassandra-exporter with Cassandra, please see the [cassandra-exporter setup docs](https://github.com/instaclustr/cassandra-exporter#usage).

To deploy and install prometheus please see the [prometheus documentation](https://prometheus.io/docs/prometheus/latest/installation/).