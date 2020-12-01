package collector

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/machinebox/progress"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ProgressFunc func(copied int64, size int64, remaining time.Duration)

type FileInfo struct {
	Path  string
	IdDir bool
}

type SSHCollectingAgent interface {
	SetTarget(host string, port int)
	SetConfig(config *ssh.ClientConfig)

	GetHost() string

	Connect() error
	ExecuteCommand(cmd string) (*bytes.Buffer, *bytes.Buffer, error)

	GetContent(path string) (*bytes.Buffer, error)
	ListDirectory(path string) ([]FileInfo, error)
	ReceiveFile(src, dest string, progressFn ProgressFunc) error
	ReceiveDir(src, dest string, progressFn ProgressFunc) error
	Remove(path string) error
}

type SSHAgent struct {
	host string
	addr string

	config *ssh.ClientConfig
	client *ssh.Client
}

func (agent *SSHAgent) SetTarget(host string, port int) {
	agent.host = host
	agent.addr = fmt.Sprintf("%s:%d", host, port)
}

func (agent *SSHAgent) SetConfig(config *ssh.ClientConfig) {
	agent.config = config
}

func (agent *SSHAgent) GetHost() string {
	return agent.host
}

func (agent *SSHAgent) Connect() error {
	client, err := ssh.Dial("tcp", agent.addr, agent.config)
	if err != nil {
		return errors.New("SSH agent: Failed to establish connection to remote host '" + agent.host + "' (" + err.Error() + ")")
	}

	agent.client = client

	return nil
}

func (agent *SSHAgent) ExecuteCommand(cmd string) (*bytes.Buffer, *bytes.Buffer, error) {
	session, err := agent.client.NewSession()
	if err != nil {
		return nil, nil, errors.New("SSH agent: Failed to create SSH session to '" + agent.host + "'")
	}
	defer session.Close()

	var outBuffer, errBuffer bytes.Buffer
	session.Stdout = &outBuffer
	session.Stderr = &errBuffer
	err = session.Run(cmd)
	if err != nil {
		return &outBuffer, &errBuffer, errors.New("SSH agent: Failed to run command '" + cmd + "' on '" + agent.host + "'. (" + err.Error() + ")")
	}

	return &outBuffer, &errBuffer, nil
}

func (agent *SSHAgent) GetContent(path string) (*bytes.Buffer, error) {
	path = filepath.Clean(path)

	client, err := sftp.NewClient(agent.client)
	if err != nil {
		return nil, errors.New("SSH agent: Failed to create SFTP session (" + err.Error() + ")")
	}
	defer client.Close()

	file, err := client.Open(path)
	if err != nil {
		return nil, errors.New("SSH agent: Failed to open file over SFTP (" + err.Error() + ")")
	}
	defer file.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(file)
	if err != nil {
		return nil, errors.New("SSH agent: Failed to read file over SFTP (" + err.Error() + ")")
	}

	return buf, nil
}

func (agent *SSHAgent) ListDirectory(path string) ([]FileInfo, error) {
	path = filepath.Clean(path)

	client, err := sftp.NewClient(agent.client)
	if err != nil {
		return nil, errors.New("SSH agent: Failed to create SFTP session (" + err.Error() + ")")
	}
	defer client.Close()

	dir, err := client.ReadDir(path)
	if err != nil {
		return nil, errors.New("SSH agent: Failed to read directory over SFTP (" + err.Error() + ")")
	}

	infos := make([]FileInfo, 0)
	for _, info := range dir {
		path := filepath.Join(path, info.Name())
		infos = append(infos, FileInfo{path, info.IsDir()})
	}

	return infos, nil
}

func (agent *SSHAgent) ReceiveFile(src, dest string, progressFn ProgressFunc) error {
	src = filepath.Clean(src)
	dest = filepath.Clean(dest)

	client, err := sftp.NewClient(agent.client)
	if err != nil {
		return errors.New("SSH agent: Failed to create SFTP session (" + err.Error() + ")")
	}
	defer client.Close()

	return agent.receiveFile(client, src, dest, progressFn)
}

func (agent *SSHAgent) receiveFile(client *sftp.Client, src, dest string, progressFn ProgressFunc) error {

	destStat, err := os.Stat(dest)
	if err != nil && !os.IsNotExist(err) {
		return errors.New("SSH agent: Failed to get information of destination file (" + err.Error() + ")")
	}
	if err == nil && destStat.IsDir() {
		dest = filepath.Join(dest, filepath.Base(src))
	}

	srcFile, err := client.Open(src)
	if err != nil {
		return errors.New("SSH agent: Failed to open source file over SFTP (" + err.Error() + ")")
	}
	defer srcFile.Close()

	var srcReader io.Reader
	srcReader = srcFile

	if progressFn != nil {
		srcStat, err := srcFile.Stat()
		if err != nil {
			return errors.New("SSH agent: Failed to open source file stat over SFTP (" + err.Error() + ")")
		}

		sourceFileSize := srcStat.Size()

		progressReader := progress.NewReader(srcFile)
		srcReader = progressReader

		go func() {
			ctx := context.Background()

			progressChan := progress.NewTicker(ctx, progressReader, sourceFileSize, 1*time.Second)

			for p := range progressChan {
				progressFn(p.N(), p.Size(), p.Remaining())
			}
		}()
	}

	destFile, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return errors.New("SSH agent: Failed to open destination file (" + err.Error() + ")")
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcReader)
	if err != nil {
		return errors.New("SSH agent: Failed to copy file over SFTP (" + err.Error() + ")")
	}

	return nil
}

func (agent *SSHAgent) ReceiveDir(src, dest string, progressFn ProgressFunc) error {
	src = filepath.Clean(src)
	dest = filepath.Clean(dest)

	err := agent.createDirectoryIfNotExists(dest)
	if err != nil {
		return err
	}

	client, err := sftp.NewClient(agent.client)
	if err != nil {
		return errors.New("SSH agent: Failed to create SFTP session (" + err.Error() + ")")
	}
	defer client.Close()

	srcStat, err := client.Stat(src)
	if err != nil {
		return errors.New("SSH agent: Failed receiver source file info over SFTP (" + err.Error() + ")")
	}

	if !srcStat.IsDir() {
		return agent.receiveFile(client, src, dest, progressFn)
	} else {
		dirSize := agent.getDirSize(client, src)

		counter := &counter{}

		if progressFn != nil {
			go func() {
				ctx := context.Background()

				progressChan := progress.NewTicker(ctx, counter, dirSize, 1*time.Second)

				for p := range progressChan {
					progressFn(p.N(), p.Size(), p.Remaining())
				}
			}()
		}

		walker := client.Walk(src)
		for walker.Step() {
			if walker.Err() != nil {
				continue
			}

			relative, err := filepath.Rel(src, walker.Path())
			if err != nil {
				return errors.New("SSH agent: Unexpected error on directory copy (" + err.Error() + ")")
			}

			if walker.Stat().IsDir() {
				err := agent.createDirectoryIfNotExists(filepath.Join(dest, relative))
				if err != nil {
					return err
				}
			} else {
				err := agent.receiveFile(client, walker.Path(), filepath.Join(dest, relative), nil)
				if err != nil {
					return err
				}

				counter.Inc(walker.Stat().Size())
			}
		}
	}

	return nil
}

func (agent *SSHAgent) getDirSize(client *sftp.Client, path string) int64 {
	var size int64 = 0
	walker := client.Walk(path)
	for walker.Step() {
		if walker.Err() != nil {
			continue
		}

		if !walker.Stat().IsDir() {
			size += walker.Stat().Size()
		}
	}

	return size
}

type counter struct {
	lock sync.RWMutex
	n    int64
	err  error
}

func (c *counter) Inc(size int64) {
	c.lock.Lock()
	c.n += size
	c.lock.Unlock()
}

func (c *counter) N() int64 {
	var n int64
	c.lock.RLock()
	n = c.n
	c.lock.RUnlock()
	return n
}

func (c *counter) Err() error {
	var err error
	c.lock.RLock()
	err = c.err
	c.lock.RUnlock()
	return err
}

func (agent *SSHAgent) createDirectoryIfNotExists(dest string) error {

	_, err := os.Stat(dest)
	if err != nil && !os.IsNotExist(err) {
		return errors.New("SSH agent: Failed to get information of destination directory (" + err.Error() + ")")
	}

	if os.IsNotExist(err) {
		err = os.MkdirAll(dest, 0777) // TODO correct permissions for directory
		if err != nil {
			return errors.New("SSH agent: Failed to create destination directory (" + err.Error() + ")")
		}
	}

	return nil
}

func (agent *SSHAgent) Remove(path string) error {
	path = filepath.Clean(path)

	client, err := sftp.NewClient(agent.client)
	if err != nil {
		return errors.New("SSH agent: Failed to create SFTP session (" + err.Error() + ")")
	}
	defer client.Close()

	stat, err := client.Stat(path)
	if err != nil {
		return errors.New("SSH agent: Failed receiver file info over SFTP (" + err.Error() + ")")
	}

	err = agent.removeRecursive(client, stat, path)
	if err != nil {
		return err
	}

	return nil
}

func (agent *SSHAgent) removeRecursive(client *sftp.Client, stat os.FileInfo, path string) error {

	if stat.IsDir() {
		err := agent.removeDir(client, path)
		if err != nil {
			return errors.New("SSH agent: Failed to remove '" + path + "' over SFTP (" + err.Error() + ")")
		}
	}

	err := client.Remove(path)
	if err != nil {
		return errors.New("SSH agent: Failed to remove '" + path + "' over SFTP (" + err.Error() + ")")
	}

	return nil
}

func (agent *SSHAgent) removeDir(client *sftp.Client, path string) error {

	dir, err := client.ReadDir(path)
	if err != nil {
		return errors.New("SSH agent: Failed to read dir '" + path + "' over SFTP (" + err.Error() + ")")
	}

	for _, info := range dir {
		err := agent.removeRecursive(client, info, filepath.Join(path, info.Name()))
		if err != nil {
			return err
		}
	}

	return nil
}
