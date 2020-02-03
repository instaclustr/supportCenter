#!/bin/bash

DATA_DIR="./data"
METRICS_PATH="$DATA_DIR/metrics/snapshot/"
METRICS_PACKAGE="$METRICS_PATH/InstaclustrCollection.tar"

if [ -z "$1" ]; then
  echo "No tarball supplied"
  echo "Usage: analyze.sh path_to_tarball"
  exit 1
fi

# Clean the data folder
rm -rf $DATA_DIR
mkdir $DATA_DIR

# Extract collected info in data folder
unzip $1 -d $DATA_DIR

if [ -f "$METRICS_PACKAGE" ]; then
  tar -vxf $METRICS_PACKAGE -C $METRICS_PATH
fi

# Start dockers
export USER_ID=$(id -u)
export GROUP_ID=$(id -g)
docker-compose up

read -r -p "Cleanup? (files, docker volumes)" response
if [[ $response =~ ^([yY][eE][sS]|[yY])$ ]]; then
  docker-compose rm -f -s -v
  rm -rf $DATA_DIR
else
  exit 0
fi
