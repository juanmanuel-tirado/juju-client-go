package utils

import (
	"errors"
	"os/exec"
	"time"

	"encoding/json"

	"github.com/juju/juju/api"
	"github.com/juju/juju/api/connector"
	"github.com/rs/zerolog/log"
)

// jujuControllerConfig is a representation of the output
// returned when running the CLI command
// `juju show-controller --show-password`
type jujuControllerConfig struct {
	ProviderDetails struct {
		UUID                   string   `json:"uuid"`
		ApiEndpoints           []string `json:"api-endpoints"`
		Cloud                  string   `json:"cloud"`
		Region                 string   `json:"region"`
		AgentVersion           string   `json:"agent-version"`
		AgentGitCommit         string   `json:"agent-git-commit"`
		ControllerModelVersion string   `json:"controller-model-version"`
		MongoVersion           string   `json:"mongo-version"`
		CAFingerprint          string   `json:"ca-fingerprint"`
		CACert                 string   `json:"ca-cert"`
	} `json:"details"`
	CurrentModel string `json:"current-model"`
	Models       map[string]struct {
		UUID      string `json:"uuid"`
		UnitCount uint   `json:"unit-count"`
	} `json:"models"`
	Account struct {
		User     string `json:"user"`
		Password string `json:"password"`
		Access   string `json:"access"`
	} `json:"account"`
}

type ControllerConfig struct {
	JujuCaCert              string
	JujuControllerAddresses []string
	JujuPassword            string
	JujuUsername            string
	ModelUUID               string
}

// connectionHelper provides functionality to stablish a connection
// with a Juju controller using the available information.
type connectionHelper struct {
	config *ControllerConfig
}

// NewConnectionHelper returns an empty connection helper.
func NewConnectionHelper() *connectionHelper {
	return &connectionHelper{}
}

// ConfigWithLocalJuju runs the locally installed juju command,
// if available, to get the current controller configuration.
func (ch *connectionHelper) ConfigWithLocalJuju() error {
	// get the value from the juju provider
	cmd := exec.Command("juju", "show-controller", "--show-password", "--format=json")

	cmdData, err := cmd.Output()
	if err != nil {
		log.Error().Err(err).Msg("error invoking juju CLI")
		return err
	}

	// given that the CLI output is a map containing arbitrary keys
	// (controllers) and fixed json structures, we have to do some
	// workaround to populate the struct
	var cliOutput interface{}
	err = json.Unmarshal(cmdData, &cliOutput)
	if err != nil {
		log.Error().Err(err).Msg("error unmarshalling Juju CLI output")
		return err
	}

	// convert it
	controllerConfig := jujuControllerConfig{}
	for _, v := range cliOutput.(map[string]interface{}) {
		// now v is a map[string]interface{} type
		marshalled, err := json.Marshal(v)
		if err != nil {
			log.Error().Err(err).Msg("error marshalling provider config")
			return err
		}
		// now we have a controllerConfig type
		err = json.Unmarshal(marshalled, &controllerConfig)
		if err != nil {
			log.Error().Err(err).Msg("error unmarshalling provider configuration from Juju CLI")
			return err
		}
		break
	}

	ch.config = &ControllerConfig{}
	ch.config.JujuCaCert = controllerConfig.ProviderDetails.CACert
	ch.config.JujuControllerAddresses = controllerConfig.ProviderDetails.ApiEndpoints
	ch.config.JujuPassword = controllerConfig.Account.Password
	ch.config.JujuUsername = controllerConfig.Account.User

	log.Debug().Interface("controllerConfig", ch.config).Msg("controller configured using juju CLI")

	return nil
}

// Connect returns a connection object not setting a target model.
func (ch *connectionHelper) Connect() (api.Connection, error) {
	return ch.ConnectWithModel("")
}

// ConnectWithModel returns a connection object using a target model UUID.
func (ch *connectionHelper) ConnectWithModel(model string) (api.Connection, error) {
	modelUUID := ""
	if model != "" {
		modelUUID = model
	}

	dialOptions := func(do *api.DialOpts) {
		//this is set as a const above, in case we need to use it elsewhere to manage connection timings
		do.Timeout = 5 * time.Minute
		//default is 2 seconds, as we are changing the overall timeout it makes sense to reduce this as well
		do.RetryDelay = 1 * time.Second
	}

	if ch.config == nil {
		err := errors.New("nil configuration")
		log.Error().Err(err).Msg("configure the helper first")
		return nil, err
	}

	connr, err := connector.NewSimple(connector.SimpleConfig{
		ControllerAddresses: ch.config.JujuControllerAddresses,
		Username:            ch.config.JujuUsername,
		Password:            ch.config.JujuPassword,
		CACert:              ch.config.JujuCaCert,
		ModelUUID:           modelUUID,
	}, dialOptions)
	if err != nil {
		return nil, err
	}

	conn, err := connr.Connect()
	if err != nil {
		return nil, err
	}

	return conn, nil
}
