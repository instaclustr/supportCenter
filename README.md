# SupportCenter
SupportCenter is a collection of tools and scripts that makes collecting information about your Cassandra cluster  and analysising it offline much easier and quicker.

* For agent documentation please see [here](docs/agent.md).
* For analysis documentation please see [here](docs/analysis.md)

## Download
You can download precompiled binaries for the agent command line tool on the [release](https://github.com/instaclustr/supportCenter/releases) page.

## Build
1. Pull anywhere where you want
2. `go build` inside `/agent` folder
```shell script
go build
```

### Run test
```shell script
go test ./... -v
```

## Notes
### Libs
```shell script
go get -u golang.org/x/crypto/ssh
```
```shell script
go get -u github.com/hnakamur/go-scp
```
### Classic build approach
```shell script
go get -u github.com/instaclustr/supportCenter/agent
```
If you did not setup global username...
```shell script
env GIT_TERMINAL_PROMPT=1 go get -u github.com/instaclustr/supportCenter/agent
```
```shell script
go install github.com/instaclustr/supportCenter/agent
```