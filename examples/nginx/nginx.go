package nginx

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type nginxContainer struct {
	*testcontainers.StartedContainer
	URI string
}

func startContainer(ctx context.Context) (*nginxContainer, error) {
	//testcontainers.NewGenericContainer(testcontainers.FromDockerfile("Dockerfile", testcontainers.OnFileSystem("."))) //source?
	//testcontainers.NewGenericContainer(testcontainers.FromDockerfile("Dockerfile", testcontainers.FromContext(tar)))
	//testcontainers.NewGenericContainer(testcontainers.FromDockerfile("Dockerfile", testcontainers.FromFS(fs))) //embedded fs

	container, err := testcontainers.Run(
		testcontainers.NewGenericContainer(
			testcontainers.FromImage("nginx", testcontainers.WithImagePlatform("linux/amd64")),
			testcontainers.WithExposedPorts("80/tcp"),
			testcontainers.WaitingFor(wait.ForLog("Server ready").WithStartupTimeout(10*time.Second)),
		),
		testcontainers.WithContext(ctx),
	)
	if err != nil {
		return nil, err
	}

	info, err := testcontainers.Info(container, testcontainers.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("http://%s:%s", info.Host(), info.MappedPort("80"))

	//startedSet = testcontainers.Up(testcontainers.NewContainerSet(c))

	return &nginxContainer{StartedContainer: container, URI: uri}, nil
}
