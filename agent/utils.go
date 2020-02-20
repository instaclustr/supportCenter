package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

type StringList struct {
	items []string
}

func (arr *StringList) String() string {
	return fmt.Sprint(arr.items)
}

func (arr *StringList) Set(value string) error {
	hosts := strings.Split(value, ",")
	for _, item := range hosts {
		arr.items = append(arr.items, item)
	}
	return nil
}

func parseAndValidateCommandLineArguments() {
	if *user == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if len(strings.TrimSpace(*mcTimeRangeFrom)) > 0 {
		timestamp, err := time.Parse(time.RFC3339, *mcTimeRangeFrom)
		if err != nil {
			log.Error("Failed to parse 'from' datetime (", *mcTimeRangeFrom, "):  ", err.Error())

			flag.PrintDefaults()
			os.Exit(1)
		}
		mcTimestampFrom = timestamp
	}

	if len(strings.TrimSpace(*mcTimeRangeTo)) > 0 {
		timestamp, err := time.Parse(time.RFC3339, *mcTimeRangeTo)
		if err != nil {
			log.Error("Failed to parse 'to' datetime: (", *mcTimeRangeFrom, "): ", err.Error())

			flag.PrintDefaults()
			os.Exit(1)
		}
		mcTimestampTo = timestamp
	}

	if mcTimestampFrom.After(mcTimestampTo) {
		log.Error("Incorrect metrics collecting time span ", mcTimestampFrom.UTC(), " after ", mcTimestampTo.UTC())

		flag.PrintDefaults()
		os.Exit(1)
	}
}

func Exists(name string) (bool, error) {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}
