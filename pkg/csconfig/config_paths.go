package csconfig

import (
	"errors"
	"path/filepath"
)

type ConfigurationPaths struct {
	ConfigDir          string `yaml:"config_dir"`
	DataDir            string `yaml:"data_dir,omitempty"`
	SimulationFilePath string `yaml:"simulation_path,omitempty"`
	HubIndexFile       string `yaml:"index_path,omitempty"` // path of the .index.json
	HubDir             string `yaml:"hub_dir,omitempty"`
	PluginDir          string `yaml:"plugin_dir,omitempty"`
	NotificationDir    string `yaml:"notification_dir,omitempty"`
	PatternDir         string `yaml:"pattern_dir,omitempty"`
}

func (c *Config) loadConfigurationPaths() error {
	if c.ConfigPaths == nil {
		return errors.New("no configuration paths provided")
	}

	if c.ConfigPaths.ConfigDir == "" {
		c.ConfigPaths.ConfigDir = filepath.Dir(c.FilePath)
	}

	if c.ConfigPaths.DataDir == "" {
		return errors.New("please provide a data directory with the 'data_dir' directive in the 'config_paths' section")
	}

	if c.ConfigPaths.HubDir == "" {
		c.ConfigPaths.HubDir = filepath.Join(c.ConfigPaths.ConfigDir, "hub")
	}

	if c.ConfigPaths.HubIndexFile == "" {
		c.ConfigPaths.HubIndexFile = filepath.Join(c.ConfigPaths.HubDir, ".index.json")
	}

	if c.ConfigPaths.NotificationDir == "" {
		c.ConfigPaths.NotificationDir = filepath.Join(c.ConfigPaths.ConfigDir, "notifications")
	}

	if c.ConfigPaths.PatternDir == "" {
		c.ConfigPaths.PatternDir = filepath.Join(c.ConfigPaths.ConfigDir, "patterns")
	}

	cleanup := []*string{
		&c.ConfigPaths.HubDir,
		&c.ConfigPaths.HubIndexFile,
		&c.ConfigPaths.ConfigDir,
		&c.ConfigPaths.DataDir,
		&c.ConfigPaths.SimulationFilePath,
		&c.ConfigPaths.PluginDir,
		&c.ConfigPaths.NotificationDir,
		&c.ConfigPaths.PatternDir,
	}

	for _, k := range cleanup {
		if err := ensureAbsolutePath(k); err != nil {
			return err
		}
	}

	return nil
}
