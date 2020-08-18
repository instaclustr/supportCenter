package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Zip(source string, target string) error {
	zipfile, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = "." + strings.TrimPrefix(path, source)

		if info.IsDir() {
			header.Name += "/"
		} else {
			if filepath.Ext(info.Name()) == ".tar" {
				header.Method = zip.Store
			} else {
				header.Method = zip.Deflate
			}
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})

	return err
}
