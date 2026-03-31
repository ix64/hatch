package core

import (
	"fmt"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ConfigOptions struct {
	ConfigFileFlag string
	DefaultName    string
	DefaultType    string
	SearchPaths    []string
}

type LoadConfigOptions[T any] struct {
	Defaults      map[string]any
	Validate      func(*T) error
	DecodeOptions []viper.DecoderConfigOption
}

func BindConfig(cmd *cobra.Command, cfgFile *string, opts ConfigOptions) {
	flagName := opts.ConfigFileFlag
	if flagName == "" {
		flagName = "config"
	}
	cmd.PersistentFlags().StringVar(cfgFile, flagName, "", "config file")
}

func NewViper(cfgFile string, opts ConfigOptions) (*viper.Viper, error) {
	vp := viper.New()
	if cfgFile != "" {
		vp.SetConfigFile(cfgFile)
	} else {
		for _, path := range opts.SearchPaths {
			vp.AddConfigPath(path)
		}
		if opts.DefaultType != "" {
			vp.SetConfigType(opts.DefaultType)
		}
		if opts.DefaultName != "" {
			vp.SetConfigName(opts.DefaultName)
		}
	}

	vp.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	vp.AutomaticEnv()
	if err := vp.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok && cfgFile == "" {
			return vp, nil
		}
		return nil, err
	}
	return vp, nil
}

func LoadConfig[T any](vp *viper.Viper, opts LoadConfigOptions[T]) (*T, error) {
	if vp == nil {
		return nil, fmt.Errorf("viper instance is required")
	}

	for key, value := range opts.Defaults {
		vp.SetDefault(key, value)
	}

	var cfg T
	decodeOptions := make([]viper.DecoderConfigOption, 0, len(opts.DecodeOptions)+1)
	decodeOptions = append(decodeOptions, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	))
	decodeOptions = append(decodeOptions, opts.DecodeOptions...)

	if err := vp.Unmarshal(&cfg, decodeOptions...); err != nil {
		return nil, err
	}
	if opts.Validate != nil {
		if err := opts.Validate(&cfg); err != nil {
			return nil, err
		}
	}

	return &cfg, nil
}
