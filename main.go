/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName(".remote-nvim")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic("failed to read config file")
	}

	start()
}

func start() {
	port := viper.GetInt("port")
	if port == 0 {
		panic("port is not set")
	} else if port > 65535 {
		panic("port is out of range")
	}

	if err := setupDevcontainer(port); err != nil {
		panic(err)
	}

	// check if devcontainer is already running
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// check if devcontainer is already running
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{Key: "label", Value: "devcontainer.local_folder=" + wd}),
	})
	if len(containers) == 0 {

		dotfilesOptions := []string{}
		if dotfilesRepository := viper.GetString("dotfilesRepository"); dotfilesRepository != "" {
			dotfilesOptions = append(dotfilesOptions, "--dotfiles-repository", dotfilesRepository)
		}
		if dotfilesTargetPath := viper.GetString("dotfilesTargetPath"); dotfilesTargetPath != "" {
			dotfilesOptions = append(dotfilesOptions, "--dotfiles-target-path", dotfilesTargetPath)
		}
		if dotfilesInstallCommand := viper.GetString("dotfilesInstallCommand"); dotfilesInstallCommand != "" {
			dotfilesOptions = append(dotfilesOptions, "--dotfiles-install-command", dotfilesInstallCommand)
		}

		if err := devcontainerUp(dotfilesOptions...); err != nil {
			panic(err)
		}

		if err := startRemoteNvim(port); err != nil {
			panic(err)
		}

		time.Sleep(1 * time.Second)
	}

	/*
	  TODO: wait for devcontainer to be ready
	*/

	if err := connectRemoteNvim(fmt.Sprintf("localhost:%d", port)); err != nil {
		panic(err)
	}
}

func setupDevcontainer(port int) error {
	if hasFile(path.Join(".devcontainer", "devcontainer.json")) {
		return nil
	}

	if err := runCmd("mkdir", "-p", ".devcontainer"); err != nil {
		return err
	}

	devcontainerJSON := DevcontainerJSON{}

	noDockerfiles := true
	if hasFile("Dockerfile") {
		devcontainerJSON.Build.Dockerfile = "Dockerfile"
		noDockerfiles = false
	} else if hasFile("docker-compose.yml") {
		devcontainerJSON.DockerComposeFile = "docker-compose.yml"
		noDockerfiles = false
	}

	if noDockerfiles {
		fmt.Print("Image with tag? (example: golang:1.20) ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		devcontainerJSON.Image = scanner.Text()
	}

	devcontainerJSON.PostCreateCommand = append(devcontainerJSON.PostCreateCommand,
		"sh", "-c", "wget -O- https://raw.githubusercontent.com/utouto97/remote-nvim/main/install-nvim.sh | sh",
	)
	devcontainerJSON.RunArgs = append(devcontainerJSON.RunArgs,
		"-e", "SSH_AUTH_SOCK=/tmp/ssh-agent.socket", "-v", "${env:SSH_AUTH_SOCK}:/tmp/ssh-agent.socket",
	)
	devcontainerJSON.AppPort = append(devcontainerJSON.AppPort, port)

	b, _ := json.MarshalIndent(devcontainerJSON, "", "  ")

	f, err := os.OpenFile(path.Join(".devcontainer", "devcontainer.json"), os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	f.Write(b)

	return nil
}

type Mount struct {
	Source string `json:"source,omitempty"`
	Target string `json:"target,omitempty"`
	Type   string `json:"type,omitempty"`
}

type DevcontainerJSON struct {
	Image string `json:"image,omitempty"`
	Build struct {
		Dockerfile string `json:"dockerfile,omitempty"`
	} `json:"build,omitempty"`
	DockerComposeFile string   `json:"dockerComposeFile,omitempty"`
	Mounts            []Mount  `json:"mounts,omitempty"`
	AppPort           []int    `json:"appPort,omitempty"`
	PostCreateCommand []string `json:"postCreateCommand,omitempty"`
	RunArgs           []string `json:"runArgs,omitempty"`
}

func hasFile(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return true
}

func hasCmd(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func devcontainerUp(opts ...string) error {
	o := append([]string{}, "up", "--workspace-folder", ".")
	o = append(o, opts...)
	if err := runCmd("devcontainer", o...); err != nil {
		return err
	}

	return nil
}

func devcontainerExec(cmd string, args ...string) error {
	as := append([]string{"exec", "--workspace-folder", ".", cmd}, args...)
	if err := runCmd("devcontainer", as...); err != nil {
		return err
	}
	return nil
}

func runCmd(cmd string, args ...string) error {
	if !hasCmd(cmd) {
		return fmt.Errorf("%s is not installed", cmd)
	}

	c := exec.Command(cmd, args...)
	c.Stdout = os.Stderr
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		return err
	}
	return nil
}

func startRemoteNvim(port int) error {
	if err := exec.Command("devcontainer", "exec", "--workspace-folder", ".", "sh", "-c",
		fmt.Sprintf("wget -O- https://raw.githubusercontent.com/utouto97/remote-nvim/main/server.sh | sh -s %d", port)).Start(); err != nil {
		return err
	}
	return nil
}

func connectRemoteNvim(address string) error {
	if !hasCmd("nvim") {
		return fmt.Errorf("nvim is not installed")
	}

	c := exec.Command("nvim", "--server", address, "--remote-ui")
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		return err
	}
	return nil
}
