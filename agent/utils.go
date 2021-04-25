package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

func initCommandLineParameters() {
	setupApplicationUsagePrinting()

	flag.Parse()
	parseAndValidateCommandLineArguments()
}

func setupApplicationUsagePrinting() {
	flag.Usage = func() {
		flagSet := flag.CommandLine

		fmt.Fprint(flagSet.Output(), "\n[Required parameters to be provided]\n")
		requiredParameters := []string{"l"}
		flagSet.VisitAll(func(flag *flag.Flag) {
			if Contains(requiredParameters, flag.Name) == true {
				printParameterUsage(flag)
			}
		})

		fmt.Fprint(flagSet.Output(), "\n[Optional parameters to be provided]\n")
		flagSet.VisitAll(func(flag *flag.Flag) {
			if Contains(requiredParameters, flag.Name) != true {
				printParameterUsage(flag)
			}
		})
	}
}

func printParameterUsage(parameter *flag.Flag) {
	s := fmt.Sprintf("  -%s", parameter.Name)
	name, usage := flag.UnquoteUsage(parameter)
	if len(name) > 0 {
		s += " " + name
	}
	if len(s) <= 4 {
		s += "\t"
	} else {
		s += "\n    \t"
	}
	s += strings.ReplaceAll(usage, "\n", "\n    \t")

	fmt.Fprintf(flag.CommandLine.Output(), "%s\n", s)
}

func parseAndValidateCommandLineArguments() {

	if *user == "" && !*generateConfig {
		flag.Usage()
		os.Exit(1)
	}

	if len(strings.TrimSpace(*mcTimeRangeFrom)) > 0 {
		timestamp, err := time.Parse(time.RFC3339, *mcTimeRangeFrom)
		if err != nil {
			log.Error("Failed to parse 'from' datetime (", *mcTimeRangeFrom, "):  ", err.Error())

			flag.Usage()
			os.Exit(1)
		}
		mcTimestampFrom = timestamp
	}

	if len(strings.TrimSpace(*mcTimeRangeTo)) > 0 {
		timestamp, err := time.Parse(time.RFC3339, *mcTimeRangeTo)
		if err != nil {
			log.Error("Failed to parse 'to' datetime: (", *mcTimeRangeFrom, "): ", err.Error())

			flag.Usage()
			os.Exit(1)
		}
		mcTimestampTo = timestamp
	}

	if mcTimestampFrom.After(mcTimestampTo) {
		log.Error("Incorrect metrics collecting time span ", mcTimestampFrom.UTC(), " after ", mcTimestampTo.UTC())

		flag.Usage()
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

func Expand(path string) string {

	if len(path) == 0 {
		return path
	}

	if path[0] != '~' {
		return path
	}

	if len(path) > 1 && path[1] != '/' {
		return path
	}

	return filepath.Join(os.Getenv("HOME"), path[1:])
}

func CopyFile(src string, dst string) error {
	from, err := os.Open(src)
	if err != nil {
		return err
	}
	defer from.Close()

	to, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		return err
	}

	return nil
}

func JoinToSet(a []string, b []string) []string {

	capacity := len(a) + len(b)
	values := make([]string, 0, capacity)
	set := make(map[string]bool, capacity)

	values = joinToSet(a, set, values)
	values = joinToSet(b, set, values)

	return values
}

func joinToSet(items []string, set map[string]bool, values []string) []string {
	for _, item := range items {
		value := strings.TrimSpace(item)
		if len(value) > 0 {
			if !set[value] {
				set[value] = true
				values = append(values, value)
			}
		}
	}
	return values
}

func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
