package conf

import (
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

//Config yaml config structure
type Config struct {
	MySQL struct {
		Host     string
		Port     int
		User     string
		Password string
		DB       string `yaml:"db"`
	}

	Pprof struct {
		CPUPprofFile string `yaml:"cpu_pprof_file"`
		MemPprofFile string `yaml:"mem_pprof_file"`
	}

	Fuse struct {
		TTL time.Duration `yaml:"ttl"`
	}
}

func LoadConfg(path string) (*Config, error) {
	conf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var c Config

	err = yaml.Unmarshal(conf, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil

}

var Conf *Config
