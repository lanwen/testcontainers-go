package testcontainers

import (
	"context"
	"log/slog"

	container2 "github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/testcontainers/testcontainers-go/wait"
)

type GenericContainerDefinition struct {
	imageSource ContainerImageSource

	exposedPorts []nat.Port

	creator    ContainerCreator
	starter    ContainerStarter
	terminator ContainerTerminator
}

type CreatedContainer struct {
	ID string

	definition GenericContainerDefinition
}

type StartedContainer struct {
	CreatedContainer
}

type GenericContainerOption func(*GenericContainerDefinition)

func WithExposedPorts(ports ...nat.Port) GenericContainerOption {
	return func(c *GenericContainerDefinition) {
		for _, port := range ports {
			c.exposedPorts = append(c.exposedPorts, port)
		}
	}
}

func WaitingFor(wait wait.Strategy) GenericContainerOption {
	return func(c *GenericContainerDefinition) {
		c.starter = &AwaitingContainerStarter{
			ContainerStarter: c.starter,
			wait:             wait,
		}
	}
}

type ContainerImageSource interface {
	Prepare(ctx context.Context) (string, error)
}

type FromImageSource struct {
	image string

	platform *v1.Platform
}

func (f *FromImageSource) Prepare(ctx context.Context) (string, error) {
	return f.image, nil
}

type FromImageSourceOption func(*FromImageSource)

func WithImagePlatform(platform string) FromImageSourceOption {
	return func(s *FromImageSource) {
		s.platform = &v1.Platform{
			Architecture: "amd64",
			OS:           "linux",
		}
	}
}

func FromImage(image string, option ...FromImageSourceOption) *FromImageSource {
	s := &FromImageSource{
		image: image,
	}

	for _, opt := range option {
		opt(s)
	}
	return s
}

type ContainerCreator interface {
	Create(ctx context.Context, definition GenericContainerDefinition) (CreatedContainer, error)
}

type ContainerStarter interface {
	Start(ctx context.Context, container CreatedContainer) (StartedContainer, error)
}

type ContainerTerminator interface {
	Terminate(ctx context.Context, container CreatedContainer) error
}

type ContainerImageCreator struct {
	client *DockerClient
}

func (c *ContainerImageCreator) Create(ctx context.Context, definition GenericContainerDefinition) (CreatedContainer, error) {
	image, err := definition.imageSource.Prepare(ctx)
	if err != nil {
		return CreatedContainer{}, err
	}
	created, err := c.client.ContainerCreate(ctx, nil, nil, nil, nil, "")
	if err != nil {
		return CreatedContainer{}, err
	}
	return CreatedContainer{
		ID: created.ID,

		definition: definition,
	}, nil
}

type LoggingContainerCreator struct {
	ContainerCreator

	logger slog.Logger
}

func (l *LoggingContainerCreator) Create(ctx context.Context, definition GenericContainerDefinition) (CreatedContainer, error) {
	l.logger.Info("Creating container")
	create, err := l.ContainerCreator.Create(ctx, definition)
	if err != nil {
		l.logger.Error("Failed to create container")
		return create, err
	}
	l.logger.Info("Created container")
	return create, err
}

type DockerContainerStarter struct {
	client *DockerClient
}

func (d *DockerContainerStarter) Start(ctx context.Context, container CreatedContainer) (StartedContainer, error) {
	err := d.client.ContainerStart(ctx, container.ID, container2.StartOptions{})
	if err != nil {
		return StartedContainer{}, err
	}
	return StartedContainer{container}, nil
}

type AwaitingContainerStarter struct {
	ContainerStarter

	wait wait.Strategy
}

func (a *AwaitingContainerStarter) Start(ctx context.Context, container CreatedContainer) (StartedContainer, error) {
	started, err := a.ContainerStarter.Start(ctx, container)
	if err != nil {
		return started, err
	}
	err = a.wait.WaitUntilReady(ctx, &Adapted{started})
	if err != nil {
		return started, err
	}
	return started, nil
}

func NewGenericContainer(source ContainerImageSource, option ...GenericContainerOption) *GenericContainerDefinition {
	return tc.NewGenericContainer(source, option...)
}

type ExecutionConfiguration struct {
	ctx context.Context
}

type ExecutionOption func(*ExecutionConfiguration)

func WithContext(ctx context.Context) ExecutionOption {
	return func(conf *ExecutionConfiguration) {
		conf.ctx = ctx
	}
}

var tc = &Testcontainers{
	client: &DockerClient{},
}

func Run(container *GenericContainerDefinition, option ...ExecutionOption) (*StartedContainer, error) {
	return tc.Run(container, option...)
}

type ContainerInfo struct {
}

func (ci ContainerInfo) Host() string {
	return ""
}

func (ci ContainerInfo) MappedPort(port nat.Port) string {
	return ""
}

func Info(container *StartedContainer, option ...ExecutionOption) (ContainerInfo, error) {
	return tc.Info(container, option...)
}

type Testcontainers struct {
	client *DockerClient

	logger slog.Logger
}

func (t *Testcontainers) NewGenericContainer(source ContainerImageSource, option ...GenericContainerOption) *GenericContainerDefinition {
	c := &GenericContainerDefinition{
		imageSource: source,

		creator: &LoggingContainerCreator{
			ContainerCreator: &ContainerImageCreator{
				client: t.client,
			},
			logger: t.logger,
		},

		starter: &DockerContainerStarter{
			client: t.client,
		},
	}

	for _, opt := range option {
		opt(c)
	}
	return c
}

func (t *Testcontainers) Run(container *GenericContainerDefinition, option ...ExecutionOption) (*StartedContainer, error) {
	conf := &ExecutionConfiguration{
		ctx: context.Background(),
	}

	for _, opt := range option {
		opt(conf)
	}

	created, err := container.creator.Create(conf.ctx, *container)
	if err != nil {
		return nil, err
	}

	started, err := container.starter.Start(conf.ctx, created)
	if err != nil {
		return nil, err
	}

	return &started, nil
}

func (t *Testcontainers) Info(container *StartedContainer, option ...ExecutionOption) (ContainerInfo, error) {
	conf := &ExecutionConfiguration{
		ctx: context.Background(),
	}

	for _, opt := range option {
		opt(conf)
	}

	return ContainerInfo{}, nil
}
