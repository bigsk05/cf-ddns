package config

type StaticConfig struct {
	// Config of system log
	Log struct {
		File struct {
			All string `mapstructure:"all"`
			Err string `mapstructure:"err"`
		} `mapstructure:"file"`
		MaxSize    int  `mapstructure:"max_size"`
		MaxBackups int  `mapstructure:"max_backups"`
		MaxAge     int  `mapstructure:"max_age"`
		Compress   bool `mapstructure:"compress"`
	} `mapstructure:"log"`
	Record struct {
		Name   string `mapstructure:"name"`
		Period int    `mapstructure:"period"`
	} `mapstructure:"record"`
	API struct {
		V4 string `mapstructure:"v4"`
		V6 string `mapstructure:"v6"`
	} `mapstructure:"api"`
	CF struct {
		Token string `mapstructure:"token"`
		Zone  string `mapstructure:"zone"`
	} `mapstructure:"cf"`
}
