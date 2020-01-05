package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type HostList struct {
	hosts []string
}

func (arr *HostList) String() string {
	return fmt.Sprint(arr.hosts)
}

func (arr *HostList) Set(value string) error {
	hosts := strings.Split(value, ",")
	for _, item := range hosts {
		arr.hosts = append(arr.hosts, item)
	}
	return nil
}

func validateCommandLineArguments() {
	if *user == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
}
