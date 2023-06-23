/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "The \"start\" subcommand launches the devcontainer with Neovim running inside.",
	Long: `The "start" subcommand initiates the devcontainer environment and automatically starts Neovim within it. This subcommand is used to quickly set up and begin working with Neovim in the devcontainer. Upon execution, it handles the necessary steps to launch the devcontainer, configure Neovim, and establish the required connections.

  The "start" subcommand ensures that the devcontainer is properly initialized and ready for use. It handles tasks such as mounting relevant directories, setting up the necessary dependencies, and establishing the connection to Neovim. Once executed, users can seamlessly start editing their files using Neovim's powerful features within the devcontainer environment.

  This subcommand is particularly useful when starting a new coding session or switching to a new development environment. It eliminates the manual setup process, allowing users to quickly dive into their coding tasks with Neovim.`,
	Run: func(cmd *cobra.Command, args []string) {
		start(args)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func start(args []string) {
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
