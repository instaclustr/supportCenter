#!/bin/bash
operatingSystems=("darwin" "linux" "windows")

for os in ${operatingSystems[@]}; do
	GOOS="${os}"
        GOARCH="amd64" 
	echo "Building $GOOS for $GOARCH"
        go build -o "agent_${os}_amd64"
	if [[ $os = "windows" ]]; then
		zip "agent_${os}_amd64.zip" "agent_${os}_amd64"
	else
		tar -czvf "agent_${os}_amd64.tar.gz" "agent_${os}_amd64"
	fi
	rm "agent_${os}_amd64"
done
