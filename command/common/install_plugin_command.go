package common

import (
	"os"

	"code.cloudfoundry.org/cli/actor/pluginaction"
	oldCmd "code.cloudfoundry.org/cli/cf/cmd"
	"code.cloudfoundry.org/cli/command"
	"code.cloudfoundry.org/cli/command/flag"
	"code.cloudfoundry.org/cli/command/plugin/shared"
	"code.cloudfoundry.org/cli/util"
	"code.cloudfoundry.org/cli/util/configv3"
)

//go:generate counterfeiter . InstallPluginActor

type InstallPluginActor interface {
	CreateExecutableCopy(path string) (string, error)
	DownloadExecutableBinaryFromURL(url string) (string, int64, error)
	FileExists(path string) bool
	GetAndValidatePlugin(metadata pluginaction.PluginMetadata, commands pluginaction.CommandList, path string) (configv3.Plugin, error)
	InstallPluginFromPath(path string, plugin configv3.Plugin) error
	IsPluginInstalled(pluginName string) bool
	UninstallPlugin(uninstaller pluginaction.PluginUninstaller, name string) error
}

type InstallPluginCommand struct {
	OptionalArgs         flag.InstallPluginArgs `positional-args:"yes"`
	Force                bool                   `short:"f" description:"Force install of plugin without confirmation"`
	RegisteredRepository string                 `short:"r" description:"Name of a registered repository where the specified plugin is located"`
	usage                interface{}            `usage:"CF_NAME install-plugin (LOCAL-PATH/TO/PLUGIN | URL | -r REPO_NAME PLUGIN_NAME) [-f]\n\nEXAMPLES:\n   CF_NAME install-plugin ~/Downloads/plugin-foobar\n   CF_NAME install-plugin https://example.com/plugin-foobar_linux_amd64\n   CF_NAME install-plugin -r My-Repo plugin-echo"`
	relatedCommands      interface{}            `related_commands:"add-plugin-repo, list-plugin-repos, plugins"`

	UI     command.UI
	Config command.Config
	Actor  InstallPluginActor
}

func (cmd *InstallPluginCommand) Setup(config command.Config, ui command.UI) error {
	cmd.UI = ui
	cmd.Config = config
	cmd.Actor = pluginaction.NewActor(config, shared.NewClient(config, ui))
	return nil
}

func (cmd InstallPluginCommand) Execute(_ []string) error {
	if !cmd.Config.Experimental() {
		oldCmd.Main(os.Getenv("CF_TRACE"), os.Args)
		return nil
	}

	var pluginNameOrLocation string
	// if cmd.RegisteredRepository != "" {
	// 	//get repo
	// } else {
	pluginNameOrLocation = cmd.OptionalArgs.PluginNameOrLocation.String()
	// }

	tempPluginPath, err := cmd.getExecutableBinary(pluginNameOrLocation)
	defer os.Remove(tempPluginPath)
	if err != nil {
		return err
	}

	rpcService, err := shared.NewRPCService(cmd.Config, cmd.UI)
	if err != nil {
		return err
	}

	plugin, err := cmd.Actor.GetAndValidatePlugin(rpcService, Commands, tempPluginPath)
	if err != nil {
		return shared.HandleError(err)
	}

	if cmd.Actor.IsPluginInstalled(plugin.Name) {
		if !cmd.Force {
			return shared.PluginAlreadyInstalledError{
				BinaryName: cmd.Config.BinaryName(),
				Name:       plugin.Name,
				Version:    plugin.Version.String(),
			}
		}

		err = cmd.uninstallPlugin(plugin, rpcService)
		if err != nil {
			return err
		}
	}

	return cmd.installPlugin(plugin, tempPluginPath)
}

func (cmd InstallPluginCommand) installPlugin(plugin configv3.Plugin, pluginPath string) error {
	cmd.UI.DisplayTextWithFlavor("Installing plugin {{.Name}}...", map[string]interface{}{
		"Name": plugin.Name,
	})

	installErr := cmd.Actor.InstallPluginFromPath(pluginPath, plugin)
	if installErr != nil {
		return installErr
	}

	cmd.UI.DisplayOK()
	cmd.UI.DisplayText("Plugin {{.Name}} {{.Version}} successfully installed.", map[string]interface{}{
		"Name":    plugin.Name,
		"Version": plugin.Version.String(),
	})
	return nil
}

func (cmd InstallPluginCommand) uninstallPlugin(plugin configv3.Plugin, rpcService *shared.RPCService) error {
	cmd.UI.DisplayText("Plugin {{.Name}} {{.Version}} is already installed. Uninstalling existing plugin...", map[string]interface{}{
		"Name":    plugin.Name,
		"Version": plugin.Version.String(),
	})

	uninstallErr := cmd.Actor.UninstallPlugin(rpcService, plugin.Name)
	if uninstallErr != nil {
		return uninstallErr
	}

	cmd.UI.DisplayOK()
	cmd.UI.DisplayText("Plugin {{.Name}} successfully uninstalled.", map[string]interface{}{
		"Name": plugin.Name,
	})

	return nil
}

func (cmd InstallPluginCommand) getExecutableBinary(pluginNameOrLocation string) (string, error) {
	var tempPath string

	switch {
	case cmd.Actor.FileExists(pluginNameOrLocation):
		err := cmd.promptForInstallPlugin(pluginNameOrLocation)
		if err != nil {
			return "", err
		}

		tempPath = pluginNameOrLocation
	case util.IsHTTPScheme(pluginNameOrLocation):
		err := cmd.promptForInstallPlugin(pluginNameOrLocation)
		if err != nil {
			return "", err
		}

		cmd.UI.DisplayText("Starting download of plugin binary from URL...")

		var size int64
		tempPath, size, err = cmd.Actor.DownloadExecutableBinaryFromURL(pluginNameOrLocation)
		defer os.Remove(tempPath)
		if err != nil {
			return "", shared.HandleError(err)
		}

		cmd.UI.DisplayText("{{.Bytes}} bytes downloaded...", map[string]interface{}{
			"Bytes": size,
		})
	case util.IsUnsupportedURLScheme(pluginNameOrLocation):
		return "", command.UnsupportedURLSchemeError{UnsupportedURL: pluginNameOrLocation}
	default:
		return "", shared.FileNotFoundError{Path: pluginNameOrLocation}
	}

	// copy twice when downloading from a URL to keep Windows specific code
	// isolated to CreateExecutableCopy
	return cmd.Actor.CreateExecutableCopy(tempPath)
}

func (cmd InstallPluginCommand) promptForInstallPlugin(path string) error {
	cmd.UI.DisplayHeader("Attention: Plugins are binaries written by potentially untrusted authors.")
	cmd.UI.DisplayHeader("Install and use plugins at your own risk.")

	if !cmd.Force {
		really, promptErr := cmd.UI.DisplayBoolPrompt(false, "Do you want to install the plugin {{.Path}}?", map[string]interface{}{
			"Path": path,
		})
		if promptErr != nil {
			return promptErr
		}
		if !really {
			return shared.PluginInstallationCancelled{}
		}
	}

	return nil
}
