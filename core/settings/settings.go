package settings

import (
	"encoding/json"
	"errors"
	"io/ioutil"
)

type networkInterface struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

type gns struct {
	networkInterface
	PKeyFilePath string `json:"pkey"`
}

type observers struct {
	Network networkInterface `json:"network"`
	GNS     gns              `json:"gns"`
}

type nodes struct {
	Network networkInterface `json:"network"`
}

type Settings struct {
	Debug     bool      `json:"debug"`
	Observers observers `json:"observers"`
	Nodes     nodes     `json:"nodes"`
}

func LoadSettings() (*Settings, error) {
	conf := &Settings{}

	bytes, err := ioutil.ReadFile("conf.json")
	if err != nil {
		return nil, errors.New("Can't read configuration. Details: " + err.Error())
	}

	err = json.Unmarshal(bytes, conf)
	if err != nil {
		return nil, errors.New("Can't read configuration. Details: " + err.Error())
	}

	return conf, nil
}
