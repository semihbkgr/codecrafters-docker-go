package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"syscall"
)

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {
	command := os.Args[3]
	args := os.Args[4:len(os.Args)]

	err := isolateFS()
	if err != nil {
		fmt.Printf("error on isolating fs: %v", err)
		os.Exit(255)
	}

	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		os.Exit(cmd.ProcessState.ExitCode())
	}
}

func isolateFS() error {
	dir, err := os.MkdirTemp("", "my_docker_fs_*")
	if err != nil {
		return err
	}

	err = os.Chmod(dir, 0777)
	if err != nil {
		return err
	}

	binPath := "/usr/local/bin"

	err = os.MkdirAll(path.Join(dir, binPath), 0777)
	if err != nil {
		return err
	}

	err = os.Link(path.Join(binPath, "docker-explorer"), path.Join(dir, binPath, "docker-explorer"))
	if err != nil {
		return err
	}

	err = syscall.Chroot(dir)
	if err != nil {
		return err
	}

	return os.Chdir("/")
}
