package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	host = flag.String("host", "", "Target hostname")
	user = flag.String("u", "", "User name")
	port = flag.Int("p", 22, "Port")
)

func validateCommandLineArguments() {
	if *host == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *user == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func main() {
	flag.Parse()
	validateCommandLineArguments()

	fmt.Println("Target host is: ", *host)
}
