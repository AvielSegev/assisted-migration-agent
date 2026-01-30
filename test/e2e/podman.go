package main

import (
	"context"
	"fmt"

	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/bindings/network"
	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/opencontainers/runtime-spec/specs-go"
	nettypes "go.podman.io/common/libnetwork/types"
)

const NetworkName = "planner"

type ContainerConfig struct {
	Image   string
	Cmd     []string
	Ports   map[int]int
	Name    string
	EnvVars map[string]string
	Volumes map[string]string
}

type PodmanRunner struct {
	conn context.Context
}

func NewPodmanRunner(socket string) (*PodmanRunner, error) {
	conn, err := bindings.NewConnection(context.Background(), socket)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to podman: %w", err)
	}
	return &PodmanRunner{conn: conn}, nil
}

func (p *PodmanRunner) StartContainer(cfg ContainerConfig) (string, error) {
	s := specgen.NewSpecGenerator(cfg.Image, false)
	s.Name = cfg.Name
	s.Command = cfg.Cmd
	s.Env = cfg.EnvVars
	s.NetNS = specgen.Namespace{NSMode: specgen.Host}

	if len(cfg.Ports) > 0 {
		s.PortMappings = make([]nettypes.PortMapping, 0, len(cfg.Ports))
		for hostPort, containerPort := range cfg.Ports {
			s.PortMappings = append(s.PortMappings, nettypes.PortMapping{
				HostPort:      uint16(hostPort),
				ContainerPort: uint16(containerPort),
				Protocol:      "tcp",
			})
		}
	}

	if len(cfg.Volumes) > 0 {
		s.Mounts = make([]specs.Mount, 0, len(cfg.Volumes))
		for hostPath, containerPath := range cfg.Volumes {
			s.Mounts = append(s.Mounts, specs.Mount{
				Type:        "bind",
				Source:      hostPath,
				Destination: containerPath,
			})
		}
	}

	createResponse, err := containers.CreateWithSpec(p.conn, s, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	if err := containers.Start(p.conn, createResponse.ID, nil); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return createResponse.ID, nil
}

func (p *PodmanRunner) WaitContainer(id string) (int32, error) {
	exitCode, err := containers.Wait(p.conn, id, nil)
	if err != nil {
		return -1, fmt.Errorf("failed to wait for container: %w", err)
	}
	return exitCode, nil
}

func (p *PodmanRunner) StopContainer(id string) error {
	if err := containers.Stop(p.conn, id, nil); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}
	return nil
}

func (p *PodmanRunner) RemoveContainer(id string) error {
	_, err := containers.Remove(p.conn, id, nil)
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}
	return nil
}

func (p *PodmanRunner) CreateNetwork() error {
	exists, err := network.Exists(p.conn, NetworkName, nil)
	if err != nil {
		return fmt.Errorf("failed to check network: %w", err)
	}
	if exists {
		return nil
	}
	_, err = network.Create(p.conn, &nettypes.Network{Name: NetworkName})
	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}
	return nil
}
