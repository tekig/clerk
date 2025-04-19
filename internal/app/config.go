package app

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/viper"
)

func readConfig(c any) error {
	configPath := flag.String("config", "", "Use corrent config")
	flag.Parse()

	viper.SetConfigType("yaml")
	if configPath == nil || *configPath == "" {
		viper.SetConfigName("config")
		viper.AddConfigPath("config")

		if err := viper.ReadInConfig(); err != nil {
			return fmt.Errorf("read config: %w", err)
		}
	} else {
		f, err := os.Open(*configPath)
		if err != nil {
			return fmt.Errorf("open `%s`: %w", *configPath, err)
		}
		defer f.Close()

		if err := viper.ReadConfig(f); err != nil {
			return fmt.Errorf("read config: %w", err)
		}
	}

	if err := viper.Unmarshal(c); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	return nil
}

func toBytes(d string) (int, error) {
	var units = map[string]int{
		"KiB": 1024,
		"MiB": 1024 * 1024,
		"GiB": 1024 * 1024 * 1024,
	}

	var split int
	for i, c := range d {
		if c >= '0' && c <= '9' {
			split = i
		}
	}

	size := d[0 : split+1]
	unit := d[1+split:]

	value, err := strconv.Atoi(size)
	if err != nil {
		return 0, fmt.Errorf("atoi `%s`: %w", size, err)
	}

	k, ok := units[unit]
	if !ok {
		return 0, fmt.Errorf("unknown unit `%s`: %w", unit, err)
	}

	return value * k, nil
}
