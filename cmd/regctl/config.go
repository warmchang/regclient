package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"

	"github.com/spf13/cobra"

	"github.com/regclient/regclient/config"
	"github.com/regclient/regclient/internal/conffile"
	"github.com/regclient/regclient/pkg/template"
)

var (
	// ConfigFilename is the default filename to read/write configuration
	ConfigFilename = "config.json"
	// ConfigDir is the default directory within the user's home directory to read/write configuration
	ConfigDir = ".regctl"
	// ConfigEnv is the environment variable to override the config filename
	ConfigEnv = "REGCTL_CONFIG"
)

// Config struct contains contents loaded from / saved to a config file
type Config struct {
	Filename      string                  `json:"-"`                 // filename that was loaded
	Version       int                     `json:"version,omitempty"` // version the file in case the config file syntax changes in the future
	Hosts         map[string]*config.Host `json:"hosts,omitempty"`
	HostDefault   *config.Host            `json:"hostDefault,omitempty"`
	BlobLimit     int64                   `json:"blobLimit,omitempty"`
	IncDockerCert *bool                   `json:"incDockerCert,omitempty"`
	IncDockerCred *bool                   `json:"incDockerCred,omitempty"`
}

type configOpts struct {
	rootOpts      *rootOpts
	blobLimit     int64
	defCredHelper string
	dockerCert    bool
	dockerCred    bool
	format        string
}

func NewConfigCmd(rOpts *rootOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config <cmd>",
		Short: "read/set configuration options",
	}
	cmd.AddCommand(newConfigGetCmd(rOpts))
	cmd.AddCommand(newConfigSetCmd(rOpts))
	return cmd
}

func newConfigGetCmd(rOpts *rootOpts) *cobra.Command {
	opts := configOpts{
		rootOpts: rOpts,
	}
	cmd := &cobra.Command{
		Use:   "get",
		Short: "show the config",
		Long:  `Displays the configuration. Passwords are not included in the output.`,
		Example: `
# retrieve the full config
regctl config get

# display the filename of the config
regctl config get --format '{{.Filename}}'`,
		Args: cobra.ExactArgs(0),
		RunE: opts.runConfigGet,
	}
	cmd.Flags().StringVar(&opts.format, "format", "{{ printPretty . }}", "format the output with Go template syntax")
	_ = cmd.RegisterFlagCompletionFunc("format", completeArgNone)
	return cmd
}

func newConfigSetCmd(rOpts *rootOpts) *cobra.Command {
	opts := configOpts{
		rootOpts: rOpts,
	}
	cmd := &cobra.Command{
		Use:   "set",
		Short: "set a configuration option",
		Long:  `Modifies an option used in future executions.`,
		Example: `
# disable loading credentials from docker
regctl config set --docker-cred=false

# enable loading credentials from docker
regctl config set --docker-cred`,
		Args: cobra.ExactArgs(0),
		RunE: opts.runConfigSet,
	}
	cmd.Flags().Int64Var(&opts.blobLimit, "blob-limit", 0, "limit for blob chunks, this is stored in memory")
	cmd.Flags().StringVar(&opts.defCredHelper, "default-cred-helper", "", "default credential helper")
	cmd.Flags().BoolVar(&opts.dockerCert, "docker-cert", false, "load certificates from docker")
	cmd.Flags().BoolVar(&opts.dockerCred, "docker-cred", false, "load credentials from docker")
	return cmd
}

func (opts *configOpts) runConfigGet(cmd *cobra.Command, args []string) error {
	c, err := ConfigLoadDefault()
	if err != nil {
		return err
	}
	for i := range c.Hosts {
		c.Hosts[i].Pass = ""
		c.Hosts[i].Token = ""
	}

	return template.Writer(cmd.OutOrStdout(), opts.format, c)
}

func (opts *configOpts) runConfigSet(cmd *cobra.Command, args []string) error {
	c, err := ConfigLoadDefault()
	if err != nil {
		return err
	}

	if flagChanged(cmd, "blob-limit") {
		c.BlobLimit = opts.blobLimit
	}
	if flagChanged(cmd, "default-cred-helper") {
		if c.HostDefault != nil {
			c.HostDefault.CredHelper = opts.defCredHelper
		}
		if c.HostDefault == nil && opts.defCredHelper != "" {
			c.HostDefault = &config.Host{
				CredHelper: opts.defCredHelper,
			}
		}
	}
	if flagChanged(cmd, "docker-cert") {
		if !opts.dockerCert {
			c.IncDockerCert = &opts.dockerCert
		} else {
			c.IncDockerCert = nil
		}
	}
	if flagChanged(cmd, "docker-cred") {
		if !opts.dockerCred {
			c.IncDockerCred = &opts.dockerCred
		} else {
			c.IncDockerCred = nil
		}
	}

	if c.HostDefault != nil && c.HostDefault.IsZero() {
		c.HostDefault = nil
	}

	err = c.ConfigSave()
	if err != nil {
		return err
	}
	return nil
}

// ConfigNew creates an empty configuration
func ConfigNew() *Config {
	c := Config{
		Hosts: map[string]*config.Host{},
	}
	return &c
}

// ConfigLoadConfFile loads the config from an io reader
func ConfigLoadConfFile(cf *conffile.File) (*Config, error) {
	r, err := cf.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	c := ConfigNew()
	if err := json.NewDecoder(r).Decode(c); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	c.Filename = cf.Name()
	// verify loaded version is not higher than supported version
	if c.Version > 1 {
		return c, ErrUnsupportedConfigVersion
	}
	for h := range c.Hosts {
		if c.Hosts[h].Name == "" {
			c.Hosts[h].Name = h
		}
		if c.Hosts[h].Hostname == "" {
			c.Hosts[h].Hostname = h
		}
		if c.Hosts[h].TLS == config.TLSUndefined {
			c.Hosts[h].TLS = config.TLSEnabled
		}
		if h == config.DockerRegistryDNS || h == config.DockerRegistry || h == config.DockerRegistryAuth {
			// Docker Hub
			c.Hosts[h].Name = config.DockerRegistry
			if c.Hosts[h].Hostname == h {
				c.Hosts[h].Hostname = config.DockerRegistryDNS
			}
			if c.Hosts[h].CredHost == h {
				c.Hosts[h].CredHost = config.DockerRegistryAuth
			}
		}
		// ensure key matches Name
		if c.Hosts[h].Name != h {
			c.Hosts[c.Hosts[h].Name] = c.Hosts[h]
			delete(c.Hosts, h)
		}
	}
	return c, nil
}

// ConfigLoadFile loads the config from a specified filename
func ConfigLoadFile(filename string) (*Config, error) {
	cf := conffile.New(conffile.WithFullname(filename))
	if cf == nil {
		return nil, fmt.Errorf("failed to define config file")
	}
	return ConfigLoadConfFile(cf)
}

// ConfigLoadDefault loads the config from the (default) filename
func ConfigLoadDefault() (*Config, error) {
	cf := conffile.New(conffile.WithDirName(ConfigDir, ConfigFilename), conffile.WithEnvFile(ConfigEnv))
	if cf == nil {
		return nil, fmt.Errorf("failed to define config file")
	}
	c, err := ConfigLoadConfFile(cf)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		// do not error on file not found
		c := ConfigNew()
		c.Filename = cf.Name()
		return c, nil
	}
	return c, err
}

// ConfigSave saves to previously loaded filename
func (c *Config) ConfigSave() error {
	cf := conffile.New(conffile.WithFullname(c.Filename))
	if cf == nil {
		return ErrNotFound
	}
	out, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	outRdr := bytes.NewReader(out)
	return cf.Write(outRdr)
}
