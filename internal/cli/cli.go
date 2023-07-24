package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	buildversion "github.com/kanopy-platform/buildah-plugin/internal/version"
	"github.com/kanopy-platform/buildah-plugin/pkg/buildah"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type RootCommand struct{}

func NewRootCommand() *cobra.Command {
	root := &RootCommand{}

	cmd := &cobra.Command{
		Use:               "buildah-plugin",
		Short:             "Plugin for CI/CD tools to run buildah commands.",
		PersistentPreRunE: root.persistentPreRunE,
		RunE:              root.runE,
	}

	cmd.PersistentFlags().String("log-level", "info", "Configure log level")
	cmd.PersistentFlags().String("access-key", "", "AWS Access Key for ECR authentication")
	cmd.PersistentFlags().String("secret-key", "", "AWS Secret Key for ECR authentication")
	cmd.PersistentFlags().String("registry", "", "ECR registry")
	cmd.PersistentFlags().String("repo", "", "The repository in the ECR registry")
	cmd.PersistentFlags().String("version", "", "JSON encoded string for version command settings")
	cmd.PersistentFlags().String("manifest", "", "JSON encoded string for manifest command settings")

	return cmd
}

func (c *RootCommand) persistentPreRunE(cmd *cobra.Command, args []string) error {
	// additional settings based on plugin type
	if err := pluginTypeSetup(); err != nil {
		return err
	}

	// bind flags to viper
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return err
	}

	// set log level
	logLevel, err := log.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		return err
	}
	log.SetLevel(logLevel)

	return nil
}

func (c *RootCommand) runE(cmd *cobra.Command, args []string) error {
	// TODO get password from AWS ECR provider

	buildah := buildah.Buildah{
		Login: buildah.Login{
			Registry: viper.GetString("registry"),
			Username: "AWS",      // TODO use output from AWS ECR provider
			Password: "password", // TODO use output from AWS ECR provider
		},
		Repo: viper.GetString("repo"),
	}

	var errs error

	if err := unmarshalIfExists("version", &buildah.Version); err != nil {
		errs = errors.Join(errs, err)
	}

	if err := unmarshalIfExists("manifest", &buildah.Manifest); err != nil {
		errs = errors.Join(errs, err)
	}

	if errs != nil {
		return errs
	}

	if buildah.Version.Print {
		log.Infof("%#v\n", buildversion.Get())
	}

	return buildah.Exec()
}

func pluginTypeSetup() error {
	pluginType := buildversion.Get().PluginType

	switch pluginType {
	case buildversion.PluginTypeDrone:
		viper.SetEnvPrefix("PLUGIN")
	default:
		return fmt.Errorf("invalid plugin type: %q", pluginType)
	}

	return nil
}

// If the value for key exists, unmarshal it into the struct v
func unmarshalIfExists(key string, v any) error {
	data := viper.GetString(key)
	if data == "" {
		return nil
	}

	return json.Unmarshal([]byte(data), v)
}
