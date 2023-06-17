/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/spf13/viper"
	"github.com/utouto97/remote-nvim/cmd"
)

func main() {
	viper.SetConfigName(".remote-nvim")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic("failed to read config file")
	}

	cmd.Execute()
}
