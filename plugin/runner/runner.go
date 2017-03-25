package runner

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	netrpc "net/rpc"

	"code.cloudfoundry.org/cli/cf/commandregistry"
	"code.cloudfoundry.org/cli/cf/configuration"
	"code.cloudfoundry.org/cli/cf/configuration/confighelpers"
	"code.cloudfoundry.org/cli/cf/configuration/pluginconfig"
	"code.cloudfoundry.org/cli/cf/trace"
	"code.cloudfoundry.org/cli/plugin/rpc"
)

type RPCServerInitializationError struct {
	Err error
}

func (e RPCServerInitializationError) Error() string {
	return fmt.Sprintf("Error initializing RPC Service: %s", e.Err)
}

type PluginNotFoundError struct {
}

func (e PluginNotFoundError) Error() string {
	return "plugin not found"
}

type Config interface {
	DialTimeout() time.Duration
	Verbose() (bool, []string)
}

type UI interface {
	DisplayWarning(template string, templateValues ...map[string]interface{})
	Writer() io.Writer
}

func RunPlugin(config Config, ui UI, args []string) error {
	isVerbose, logFiles := config.Verbose()
	traceLogger := trace.NewLogger(ui.Writer(), isVerbose, logFiles...)

	deps := commandregistry.NewDependency(ui.Writer(), traceLogger, fmt.Sprint(config.DialTimeout().Seconds()))
	defer deps.Config.Close()

	server := netrpc.NewServer()
	rpcService, err := rpc.NewRpcService(deps.TeePrinter, deps.TeePrinter, deps.Config, deps.RepoLocator, rpc.NewCommandRunner(), deps.Logger, ui.Writer(), server)
	if err != nil {
		return RPCServerInitializationError{Err: err}
	}

	pluginPath := filepath.Join(confighelpers.PluginRepoDir(), ".cf", "plugins")
	pluginConfig := pluginconfig.NewPluginConfig(
		func(err error) {
			ui.DisplayWarning("Error read/writing plugin config: {{.ErrorMessage}}", map[string]interface{}{
				"ErrorMessage": err.Error(),
			})
		},
		configuration.NewDiskPersistor(filepath.Join(pluginPath, "config.json")),
		pluginPath,
	)
	pluginList := pluginConfig.Plugins()

	ran := rpc.RunMethodIfExists(rpcService, args, pluginList)
	if !ran {
		return PluginNotFoundError{}
	}
	return nil
}
