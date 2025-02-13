// Copyright 2020 Nokia
// Licensed under the BSD 3-Clause License.
// SPDX-License-Identifier: BSD-3-Clause

package sonic

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/srl-labs/containerlab/nodes"
	"github.com/srl-labs/containerlab/types"
	"github.com/srl-labs/containerlab/utils"
)

var kindnames = []string{"sonic-vs"}

func init() {
	nodes.Register(kindnames, func() nodes.Node {
		return new(sonic)
	})
}

type sonic struct {
	nodes.DefaultNode
}

func (s *sonic) Init(cfg *types.NodeConfig, opts ...nodes.NodeOption) error {
	s.Cfg = cfg
	for _, o := range opts {
		o(s)
	}
	// the entrypoint is reset to prevent it from starting before all interfaces are connected
	// all main sonic agents are started in a post-deploy phase
	s.Cfg.Entrypoint = "/bin/bash"
	return nil
}

func (s *sonic) PreDeploy(_, _, _ string) error {
	utils.CreateDirectory(s.Cfg.LabDir, 0777)
	return nil
}

func (s *sonic) PostDeploy(ctx context.Context, _ map[string]nodes.Node) error {
	log.Debugf("Running postdeploy actions for sonic-vs '%s' node", s.Cfg.ShortName)

	err := s.Runtime.ExecNotWait(ctx, s.Cfg.ContainerID, []string{"supervisord"})
	if err != nil {
		return fmt.Errorf("failed post-deploy node %q: %w", s.Cfg.ShortName, err)
	}

	err = s.Runtime.ExecNotWait(ctx, s.Cfg.ContainerID, []string{"supervisorctl start bgpd"})
	if err != nil {
		return fmt.Errorf("failed post-deploy node %q: %w", s.Cfg.ShortName, err)
	}

	return nil
}
