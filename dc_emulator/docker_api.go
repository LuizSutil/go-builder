package dc_emulator

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"
)

type dockerRunArgs struct {
	build    string
	image    string
	env_file []string
	volumes  []string
	command  []string
}

func DockerApi(container_name string, args dockerRunArgs) {

	err := createContainer(container_name, args)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func createContainer(container_name string, args dockerRunArgs) error {

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	_, err = buildDockerImage(context.Background(), args.build, container_name)
	if err != nil {
		return err
	}

	// Gets variables from env file, should make into a method to
	envFile := args.env_file[0]
	envFileContent, err := os.ReadFile(envFile)
	if err != nil {
		return err
	}
	envVariables := strings.Split(string(envFileContent), "\n")

	containerConfig := &container.Config{
		Image: container_name,
		Env:   envVariables,
		Cmd:   args.command,
	}

	hostConfig := &container.HostConfig{
		Binds: args.volumes,
	}

	resp, err := cli.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		nil,
		nil,
		container_name,
	)
	if err != nil {
		return err
	}

	err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})
		return err
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})
			panic(err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})
		panic(err)
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})
	return nil
}

func buildDockerImage(ctx context.Context, dockerfilePath, imageName string) (string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", err
	}

	buildContext, err := archive.TarWithOptions(dockerfilePath, &archive.TarOptions{})
	if err != nil {
		return "", err
	}

	buildResponse, err := cli.ImageBuild(
		ctx,
		buildContext,
		types.ImageBuildOptions{
			Dockerfile: "Dockerfile", // Path to the Dockerfile in the build context
			Tags:       []string{imageName},
		},
	)
	if err != nil {
		return "", err
	}
	defer buildResponse.Body.Close()

	// Used to print out the commands of the building process
	_, err = io.Copy(os.Stdout, buildResponse.Body)
	if err != nil {
		return "", err
	}

	args := filters.NewArgs()
	args.Add("reference", imageName)

	listOptions := types.ImageListOptions{
		Filters: args,
	}

	dc, _ := cli.ImageList(ctx, listOptions)

	imageID := strings.TrimPrefix(dc[0].ID, "åsha256:")
	return imageID, nil
}
