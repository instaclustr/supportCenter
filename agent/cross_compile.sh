#!/bin/bash
operatingSystems=("darwin" "linux" "windows")

for os in ${operatingSystems[@]}; do
	echo "Building $os for amd64"
	if [[ $os = "windows" ]]; then
		GOOS="${os}" GOARCH="amd64" go build -o "agent_${os}_amd64.exe"
		zip "agent_${os}_amd64.zip" "agent_${os}_amd64.exe"
	else
	        GOOS="${os}" GOARCH="amd64" go build -o "agent_${os}_amd64"
		tar -czvf "agent_${os}_amd64.tar.gz" "agent_${os}_amd64"
	fi
#	rm "agent_${os}_amd64"
done
