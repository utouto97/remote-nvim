/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/spf13/cobra"
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
  if hasFile(path.Join(".devcontainer", "devcontainer.json")) {
    return
  }

  if err := exec.Command("mkdir", "-p", ".devcontainer").Run(); err != nil {
    panic(err)
  }

  devcontainerJSON := DevcontainerJSON{}

  noDockerfiles := true
  if hasFile("Dockerfile") {
    devcontainerJSON.Build.Dockerfile = "Dockerfile"
  } else if hasFile("docker-compose.yml") {
    devcontainerJSON.DockerComposeFile = "docker-compose.yml"
  }

  if noDockerfiles {
    fmt.Print("image? ")
    scanner := bufio.NewScanner(os.Stdin)
    scanner.Scan()
    devcontainerJSON.Image = scanner.Text()
  }

  b, _ := json.Marshal(devcontainerJSON)

  f, err := os.OpenFile(path.Join(".devcontainer", "devcontainer.json"), os.O_WRONLY|os.O_CREATE, 0666)
  if err != nil {
    panic(err)
  }
  f.Write(b)
}

type DevcontainerJSON struct {
  Image string `json:"image,omitempty"`
  Build struct {
    Dockerfile string `json:"dockerfile,omitempty"`
  } `json:"build,omitempty"`
  DockerComposeFile string `json:"dockerComposeFile,omitempty"`
}

func hasFile(filename string) bool {
  if _, err := os.Stat(filename); os.IsNotExist(err) {
    return false
  }
  return true
}

