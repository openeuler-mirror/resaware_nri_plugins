package policy

import (
	"github.com/containerd/nri/pkg/api"
)

type Config struct {
	needUpdate []*api.ContainerUpdate
}

func (c *Config) Push(u *api.ContainerUpdate) {
	if c.needUpdate == nil {
		c.needUpdate = make([]*api.ContainerUpdate, 0)
	}
	c.needUpdate = append(c.needUpdate, u)
}

func (c *Config) GetUpdate() []*api.ContainerUpdate {
	return c.needUpdate
}

func (c *Config) Validate() error {
	return nil
}
