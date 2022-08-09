package main

import (
	"os"

	"github.com/juju/juju-client-go/utils"
	"github.com/juju/juju/api/client/application"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func doSomething() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	helper := utils.NewConnectionHelper()
	if err := helper.ConfigWithLocalJuju(); err != nil {
		log.Error().Err(err).Msg("failed when configuring connection")
	}

	modelUUID := "f72ef260-3f4d-4f29-8e2a-32fc2bbfea60"
	conn, err := helper.ConnectWithModel(modelUUID)
	if err != nil {
		return
	}

	applicationClient := application.NewClient(conn)
	defer applicationClient.Close()
	config, err := applicationClient.Get("master", "tiny-bash")
	if err != nil {
		log.Error().Err(err).Msg("error getting configuration")
		return
	}
	log.Info().Interface("config", config).Msg("tiny-bash config")
}

func main() {
	doSomething()
}
