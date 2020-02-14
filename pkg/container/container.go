package container

import (
	"fmt"
	"os"

	dockerapi "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/pkg/term"
	"github.com/oclaussen/dodo/pkg/types"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

const (
	DefaultAPIVersion = "1.39"
)

type Container struct {
	name        string
	daemon      bool
	config      *types.Backdrop
	client      *client.Client
	context     context.Context
	tmpPath     string
	authConfigs map[string]dockerapi.AuthConfig
}

func NewContainer(config *types.Backdrop, authConfigs map[string]dockerapi.AuthConfig, daemon bool) (*Container, error) {
	dockerClient, err := getDockerClient()
	if err != nil {
		return nil, err
	}

	name := config.ContainerName
	if daemon {
		name = config.Name
	} else if len(name) == 0 {
		name = fmt.Sprintf("%s-%s", config.Name, stringid.GenerateRandomID()[:8])
	}

	return &Container{
		name:        name,
		daemon:      daemon,
		config:      config,
		client:      dockerClient,
		context:     context.Background(),
		tmpPath:     fmt.Sprintf("/tmp/dodo-%s/", stringid.GenerateRandomID()[:20]),
		authConfigs: authConfigs,
	}, nil
}

func (c *Container) Run() error {
	imageId, err := c.GetImage()
	if err != nil {
		return err
	}

	containerID, err := c.create(imageId)
	if err != nil {
		return err
	}

	if c.daemon {
		return c.client.ContainerStart(
			c.context,
			containerID,
			dockerapi.ContainerStartOptions{},
		)
	} else {
		return c.run(containerID, hasTTY())
	}
}

func (c *Container) Stop() error {
	if err := c.client.ContainerStop(c.context, c.name, nil); err != nil {
		return err
	}

	if err := c.client.ContainerRemove(c.context, c.name, dockerapi.ContainerRemoveOptions{}); err != nil {
		return err
	}

	return nil
}

func hasTTY() bool {
	_, inTerm := term.GetFdInfo(os.Stdin)
	_, outTerm := term.GetFdInfo(os.Stdout)
	return inTerm && outTerm
}

func getDockerClient() (*client.Client, error) {
	opts := &configuration.ClientOptions{}
	mutators := []client.Opt{}
	if len(opts.Version) > 0 {
		mutators = append(mutators, client.WithVersion(opts.Version))
	} else {
		mutators = append(mutators, client.WithVersion(DefaultAPIVersion))
	}
	if len(opts.Host) > 0 {
		mutators = append(mutators, client.WithHost(opts.Host))
	}
	if len(opts.CAFile)+len(opts.CertFile)+len(opts.KeyFile) > 0 {
		mutators = append(mutators, client.WithTLSClientConfig(opts.CAFile, opts.CertFile, opts.KeyFile))
	}
	return client.NewClientWithOpts(mutators...)
}
