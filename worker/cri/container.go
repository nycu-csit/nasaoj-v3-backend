package cri

import (
	"context"
	"fmt"
	"time"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type ContainerConfig struct {
	Name        string
	Image       string
	Command     []string
	Args        []string
	WorkingDir  string
	Mounts      []Mount
	Envs        map[string]string
	Labels      map[string]string
	Annotations map[string]string
}

type Mount struct {
	HostPath      string
	ContainerPath string
	ReadOnly      bool
}

type ContainerResult struct {
	ExitCode int32
	Stdout   string
	Stderr   string
}

func (c *Client) CreateAndRunContainer(
	ctx context.Context, config *ContainerConfig,
) (*ContainerResult, error) {
	podSandboxConfig := &runtimeapi.PodSandboxConfig{
		Metadata: &runtimeapi.PodSandboxMetadata{
			Name:      config.Name + "-pod",
			Namespace: "nasaoj",
			Uid:       config.Name,
		},
		Labels:      config.Labels,
		Annotations: config.Annotations,
	}

	podResp, err := c.runtimeService.RunPodSandbox(ctx,
		&runtimeapi.RunPodSandboxRequest{
			Config: podSandboxConfig,
		})
	if err != nil {
		return nil, fmt.Errorf("failed to create pod sandbox: %w", err)
	}
	podSandboxId := podResp.PodSandboxId

	defer func() {
		_, _ = c.runtimeService.RemovePodSandbox(context.Background(),
			&runtimeapi.RemovePodSandboxRequest{
				PodSandboxId: podSandboxId,
			})
	}()

	var mounts []*runtimeapi.Mount
	for _, m := range config.Mounts {
		mounts = append(mounts, &runtimeapi.Mount{
			HostPath:      m.HostPath,
			ContainerPath: m.ContainerPath,
			Readonly:      m.ReadOnly,
		})
	}

	var envs []*runtimeapi.KeyValue
	for k, v := range config.Envs {
		envs = append(envs, &runtimeapi.KeyValue{
			Key:   k,
			Value: v,
		})
	}

	containerConfig := &runtimeapi.ContainerConfig{
		Metadata: &runtimeapi.ContainerMetadata{
			Name: config.Name,
		},
		Image: &runtimeapi.ImageSpec{
			Image: config.Image,
		},
		Command:     config.Command,
		Args:        config.Args,
		WorkingDir:  config.WorkingDir,
		Mounts:      mounts,
		Envs:        envs,
		Labels:      config.Labels,
		Annotations: config.Annotations,
	}

	createResp, err := c.runtimeService.CreateContainer(ctx,
		&runtimeapi.CreateContainerRequest{
			PodSandboxId:  podSandboxId,
			Config:        containerConfig,
			SandboxConfig: podSandboxConfig,
		})
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}
	containerId := createResp.ContainerId

	defer func() {
		_, _ = c.runtimeService.RemoveContainer(context.Background(),
			&runtimeapi.RemoveContainerRequest{
				ContainerId: containerId,
			})
	}()

	_, err = c.runtimeService.StartContainer(ctx, &runtimeapi.StartContainerRequest{
		ContainerId: containerId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	result, err := c.waitForContainer(ctx, containerId)
	if err != nil {
		_, _ = c.runtimeService.StopContainer(context.Background(),
			&runtimeapi.StopContainerRequest{
				ContainerId: containerId,
				Timeout:     0,
			})
		return nil, err
	}

	return result, nil
}

func (c *Client) waitForContainer(
	ctx context.Context, containerId string,
) (*ContainerResult, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			statusResp, err := c.runtimeService.ContainerStatus(ctx,
				&runtimeapi.ContainerStatusRequest{
					ContainerId: containerId,
					Verbose:     true,
				})
			if err != nil {
				return nil, fmt.Errorf("failed to get container status: %w", err)
			}

			status := statusResp.Status
			if status.State == runtimeapi.ContainerState_CONTAINER_EXITED {
				return &ContainerResult{
					ExitCode: status.ExitCode,
				}, nil
			}
		}
	}
}
