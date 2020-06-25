package main

import (
	"context"
	"encoding/base64"
	"flag"
	"github.com/rancher/remotedialer"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"time"
)

var (
	configFile string
)

func init() {
	flag.StringVar(&configFile, "config", "/etc/cmdplus/tunnel-agent.yml", "Path to cmdplus-tunnel configuration file")
	flag.Parse()

	//remotedialer.PrintTunnelData = true
	//logrus.SetLevel(logrus.DebugLevel)
}

func main() {
	conf := new(configure)
	confData, err := ioutil.ReadFile(configFile)
	if err != nil {
		logrus.Fatalf("read file %s failed: %v", configFile, err)
	}
	err = yaml.Unmarshal(confData, conf)
	if err != nil {
		logrus.Fatalf("decode yaml failed: %v", err)
	}

	wsURL := conf.TunnelURL
	if wsURL == "" {
		wsURL = "wss://tunnel.cmd.plus/tunnel"
	}
	headers := http.Header{}
	headers.Set("Authorization", "Basic "+basicAuth(conf.AgentId, conf.AgentToken))

	// retry while disconnect
	for {
		remotedialer.ClientConnect(
			context.Background(),
			wsURL, headers, nil,
			func(string, string) bool { return true },
			nil,
		)
		time.Sleep(5 * time.Second)
	}
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

type configure struct {
	AgentId    string `yaml:"id"`
	AgentToken string `yaml:"token"`
	TunnelURL  string `yaml:"tunnel_url"`
}
