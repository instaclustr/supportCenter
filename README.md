# SupportCenter
SupportCenter private development repo

## Build
### Import
```shell script
go get -u github.com/instaclustr/supportCenter/agent
```
If you did not setup global username...
```shell script
env GIT_TERMINAL_PROMPT=1 go get -u github.com/instaclustr/supportCenter/agent
```

### Build
```shell script
go install github.com/instaclustr/supportCenter/agent
```

## Notes
### Libs
```shell script
go get -u golang.org/x/crypto/ssh
```
```shell script
go get -u github.com/hnakamur/go-scp
```
