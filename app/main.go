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
	image := os.Args[2]
	imgDir, err := PullImage(image, "./images")
	if err != nil {
		fmt.Printf("error on pulling image: %v", err)
		os.Exit(255)
	}

	command := os.Args[3]
	args := os.Args[4:len(os.Args)]

	//fsDir, err := isolatedFS()
	//if err != nil {
	//	fmt.Printf("error on isolating fs: %v", err)
	//	os.Exit(255)
	//}

	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot:     imgDir,
		Cloneflags: syscall.CLONE_NEWPID,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(cmd.ProcessState.ExitCode())
	}
}

func isolatedFS() (string, error) {
	dir, err := os.MkdirTemp("", "my_docker_fs_*")
	if err != nil {
		return "", err
	}

	err = os.Chmod(dir, 0777)
	if err != nil {
		return "", err
	}

	binPath := "/usr/local/bin"

	err = os.MkdirAll(path.Join(dir, binPath), 0777)
	if err != nil {
		return "", err
	}

	err = os.Link(path.Join(binPath, "docker-explorer"), path.Join(dir, binPath, "docker-explorer"))
	if err != nil {
		return "", err
	}

	return dir, nil
}
