// Copyright 2020 Nokia
// Licensed under the BSD 3-Clause License.
// SPDX-License-Identifier: BSD-3-Clause

package vr_csr

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/srl-labs/containerlab/netconf"
	"github.com/srl-labs/containerlab/nodes"
	"github.com/srl-labs/containerlab/types"
	"github.com/srl-labs/containerlab/utils"
)

var kindnames = []string{"vr-csr", "vr-cisco_csr1000v"}

const (
	scrapliPlatformName = "cisco_iosxe"

	configDirName   = "config"
	startupCfgFName = "startup-config.cfg"

	defaultUser     = "admin"
	defaultPassword = "admin"
)

func init() {
	nodes.Register(kindnames, func() nodes.Node {
		return new(vrCsr)
	})
	err := nodes.SetDefaultCredentials(kindnames, defaultUser, defaultPassword)
	if err != nil {
		log.Error(err)
	}
}

type vrCsr struct {
	nodes.DefaultNode
}

func (s *vrCsr) Init(cfg *types.NodeConfig, opts ...nodes.NodeOption) error {
	s.Cfg = cfg
	for _, o := range opts {
		o(s)
	}
	// env vars are used to set launch.py arguments in vrnetlab container
	defEnv := map[string]string{
		"CONNECTION_MODE":    nodes.VrDefConnMode,
		"USERNAME":           "admin",
		"PASSWORD":           "admin",
		"DOCKER_NET_V4_ADDR": s.Mgmt.IPv4Subnet,
		"DOCKER_NET_V6_ADDR": s.Mgmt.IPv6Subnet,
	}
	s.Cfg.Env = utils.MergeStringMaps(defEnv, s.Cfg.Env)

	// mount config dir to support startup-config functionality
	s.Cfg.Binds = append(s.Cfg.Binds, fmt.Sprint(path.Join(s.Cfg.LabDir, configDirName), ":/config"))

	if s.Cfg.Env["CONNECTION_MODE"] == "macvtap" {
		// mount dev dir to enable macvtap
		s.Cfg.Binds = append(s.Cfg.Binds, "/dev:/dev")
	}

	s.Cfg.Cmd = fmt.Sprintf("--username %s --password %s --hostname %s --connection-mode %s --trace",
		s.Cfg.Env["USERNAME"], s.Cfg.Env["PASSWORD"], s.Cfg.ShortName, s.Cfg.Env["CONNECTION_MODE"])

	// set virtualization requirement
	s.Cfg.HostRequirements.VirtRequired = true

	return nil
}

func (s *vrCsr) PreDeploy(_, _, _ string) error {
	utils.CreateDirectory(s.Cfg.LabDir, 0777)
	return loadStartupConfigFile(s.Cfg)
}

func (s *vrCsr) SaveConfig(_ context.Context) error {
	err := netconf.SaveConfig(s.Cfg.LongName,
		defaultUser,
		defaultPassword,
		scrapliPlatformName,
	)
	if err != nil {
		return err
	}

	log.Infof("saved %s running configuration to startup configuration file\n", s.Cfg.ShortName)
	return nil
}

func loadStartupConfigFile(node *types.NodeConfig) error {
	// create config directory that will be bind mounted to vrnetlab container at / path
	utils.CreateDirectory(path.Join(node.LabDir, configDirName), 0777)

	if node.StartupConfig != "" {
		// dstCfg is a path to a file on the clab host that will have rendered configuration
		dstCfg := filepath.Join(node.LabDir, configDirName, startupCfgFName)

		c, err := os.ReadFile(node.StartupConfig)
		if err != nil {
			return err
		}

		cfgTemplate := string(c)

		err = node.GenerateConfig(dstCfg, cfgTemplate)
		if err != nil {
			log.Errorf("node=%s, failed to generate config: %v", node.ShortName, err)
		}
	}
	return nil
}
