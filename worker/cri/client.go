package cri

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type Client struct {
	conn           *grpc.ClientConn
	runtimeService runtimeapi.RuntimeServiceClient
	imageService   runtimeapi.ImageServiceClient
}

func NewClient(endpoint string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to CRI endpoint %s: %w", endpoint, err)
	}

	return &Client{
		conn:           conn,
		runtimeService: runtimeapi.NewRuntimeServiceClient(conn),
		imageService:   runtimeapi.NewImageServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) RuntimeService() runtimeapi.RuntimeServiceClient {
	return c.runtimeService
}

func (c *Client) ImageService() runtimeapi.ImageServiceClient {
	return c.imageService
}

// StopContainer stops a container by name with a grace period timeout.
// The timeout specifies the number of seconds to wait before forcefully killing.
func (c *Client) StopContainer(ctx context.Context, name string, timeout int64) error {
	// List containers to find the one with matching name
	listResp, err := c.runtimeService.ListContainers(ctx,
		&runtimeapi.ListContainersRequest{
			Filter: &runtimeapi.ContainerFilter{
				LabelSelector: map[string]string{
					"nasaoj.job.id": name,
				},
			},
		})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	for _, container := range listResp.Containers {
		if container.State == runtimeapi.ContainerState_CONTAINER_RUNNING {
			_, err := c.runtimeService.StopContainer(ctx,
				&runtimeapi.StopContainerRequest{
					ContainerId: container.Id,
					Timeout:     timeout,
				})
			if err != nil {
				return fmt.Errorf("failed to stop container %s: %w", container.Id, err)
			}
		}
	}

	return nil
}
