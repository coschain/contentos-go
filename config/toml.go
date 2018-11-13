package config

import (
	"bytes"
	"github.com/coschain/contentos-go/node"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"
)

var configTemplate *template.Template

const DefaultConfigTemplate = `# This is a TOML config file. 
# For more information, see https://github.com/toml-lang/toml

DataDir = "{{ .DataDir }}"

[timer]

Interval = "{{ .Timer.Interval }}"

[grpc]

RPCListen = "{{ .GRPC.RPCListen }}"
HTTPListen = "{{ .GRPC.HTTPListen }}"
HTTPCors = "{{ .GRPC.HTTPCors }}"
HTTPLimit = {{ .GRPC.HTTPLimit }}
`

func WriteNodeConfigFile(configDirPath string, configName string, config node.Config, mode os.FileMode) error {
	var buffer bytes.Buffer
	var err error

	if configTemplate, err = template.New("configFileTemplate").Parse(DefaultConfigTemplate); err != nil {
		return err
	}

	if err = configTemplate.Execute(&buffer, config); err != nil {
		return err
	}
	configPath := filepath.Join(configDirPath, configName)
	return ioutil.WriteFile(configPath, buffer.Bytes(), mode)
}
