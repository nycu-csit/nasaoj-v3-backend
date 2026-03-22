package cri

import (
	"context"
	"fmt"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

func (c *Client) PullImage(ctx context.Context, image string) error {
	_, err := c.imageService.PullImage(ctx, &runtimeapi.PullImageRequest{
		Image: &runtimeapi.ImageSpec{
			Image: image,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", image, err)
	}
	return nil
}

func (c *Client) ImageExists(ctx context.Context, image string) (bool, error) {
	resp, err := c.imageService.ImageStatus(ctx, &runtimeapi.ImageStatusRequest{
		Image: &runtimeapi.ImageSpec{
			Image: image,
		},
	})
	if err != nil {
		return false, fmt.Errorf("failed to check image status %s: %w", image, err)
	}
	return resp.Image != nil, nil
}
