# SupportCenter
SupportCenter private development repo

## Build
1. Pull anywhere where you want
2. `go build` inside `/agent` folder
```shell script
go build
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