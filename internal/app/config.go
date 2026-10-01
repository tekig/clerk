package app

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

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
	units := map[string]int{
		"KiB": 1024,
		"MiB": 1024 * 1024,
		"GiB": 1024 * 1024 * 1024,
	}

	// Leading digits are the size, the rest is the unit
	split := strings.IndexFunc(d, func(c rune) bool {
		return c < '0' || c > '9'
	})
	if split == -1 {
		split = len(d)
	}

	size := d[:split]
	unit := d[split:]

	value, err := strconv.Atoi(size)
	if err != nil {
		return 0, fmt.Errorf("atoi `%s`: %w", size, err)
	}

	k, ok := units[unit]
	if !ok {
		return 0, fmt.Errorf("unknown unit `%s`", unit)
	}

	return value * k, nil
}
