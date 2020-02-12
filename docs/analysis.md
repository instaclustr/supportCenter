# Analysis
The Instaclustr collection agent comes with an analysis tool that makes working with agent generated tarballs a lot easier. It comes with a simple bash script that will extract the contents of the support tarball and run a number of docker containers that contain elasticsearch, prometheus and grafana. 

The containers are configured to ingest the provided logs and load the prometheus snapshots. Grafana is also configured with a set of dashboards usefull for working with Cassandra metrics.

## Quickstart
Run the `./analyze.sh MYSUPPORTTARBALL.tar` script from within the analysis directory. The tarball can be in any location, but you must run the analyze script from within the analysis directory. For example:

```shell script
cd ./analysis
./analyze.sh path/to/collected/tarball/20200128T200545-data.zip
```

The script will also start in the foreground and tail all docker logs, to shutdown the containers and clean up your analysis environment. Simply send the exit signal (`ctrl-c` on most terminal environments). The script will offer to clean up the extracted tarball and containers.

While running the analysis script, you can find other extracted information in the data directory (automatically created) under the `nodes/IP/*`.

For example the `jvm.options` file for the node 123.123.123.1 will be in `data/123.123.123.1/config` and the output of `nodetool cfstats` will be in `data/123.123.123.1/info/nodetool_cfstats_-H.info`.

## Requirements
To run the analysis environment, you will need the following:
* bash compatible shell environment
* unzip (command line zip)
* docker (and docker-compose support)
* a browser of some description

## Customization
Currently you can customise the `analyze.sh` to suit your environment. The main variables you may wish to change are below:

```shell script
DATA_DIR="./data"
METRICS_PATH="$DATA_DIR/metrics/snapshot/"
METRICS_PACKAGE="$METRICS_PATH/InstaclustrCollection.tar"
```