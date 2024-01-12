package dc_emulator

import (
	"context"
	"errors"
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

func returnErroHelper(errorList ...error) error {
	if err := errors.Join(errorList...); err != nil {
		return err
	}
	return nil
}

func DockerApi(args dockerRunArgs) {

	ctx := context.Background()
	cli, clientErr := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	// Need to look for image and not rebuild if exists
	imagetag := fmt.Sprintf("%s:latest", args.image)
	imageExists, checkImageErr := imageExists(ctx, cli, imagetag)

	var buildErr error
	if !imageExists {
		_, buildErr = buildDockerImage(ctx, args)

	}
	containerId, createContainerErr := createContainer(cli, ctx, args)

	runDockerErr := runDockerContainer(cli, ctx, containerId)

	err := returnErroHelper(
		buildErr,
		clientErr,
		checkImageErr,
		createContainerErr,
		runDockerErr,
	)
	if err != nil {
		fmt.Println(err)
	}
}

func createContainer(cli *client.Client, ctx context.Context, args dockerRunArgs) (string, error) {

	// Gets variables from env file, should make into a method to
	envFile := args.env_file[0]
	envFileContent, envFileReadErr := os.ReadFile(envFile)

	envVariables := strings.Split(string(envFileContent), "\n")

	containerConfig := &container.Config{
		Image: args.image,
		Env:   envVariables,
		Cmd:   args.command,
	}

	hostConfig := &container.HostConfig{
		Binds: args.volumes,
	}

	resp, createContainerErr := cli.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		nil,
		nil,
		args.image,
	)

	err := returnErroHelper(
		envFileReadErr,
		createContainerErr,
	)

	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func runDockerContainer(cli *client.Client, ctx context.Context, containerId string) error {

	err := cli.ContainerStart(ctx, containerId, types.ContainerStartOptions{})
	if err != nil {
		cli.ContainerRemove(ctx, containerId, types.ContainerRemoveOptions{})
		return err
	}

	statusCh, errCh := cli.ContainerWait(ctx, containerId, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			cli.ContainerRemove(ctx, containerId, types.ContainerRemoveOptions{})
			panic(err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, containerId, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		cli.ContainerRemove(ctx, containerId, types.ContainerRemoveOptions{})
		panic(err)
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	cli.ContainerRemove(ctx, containerId, types.ContainerRemoveOptions{})

	return nil
}

func buildDockerImage(ctx context.Context, args dockerRunArgs) (string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", err
	}

	buildContext, err := archive.TarWithOptions(args.build, &archive.TarOptions{})
	if err != nil {
		return "", err
	}

	buildResponse, err := cli.ImageBuild(
		ctx,
		buildContext,
		types.ImageBuildOptions{
			Dockerfile: "Dockerfile", // Path to the Dockerfile in the build context
			Tags:       []string{args.image},
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

	args_ref := filters.NewArgs()
	args_ref.Add("reference", args.image)

	listOptions := types.ImageListOptions{
		Filters: args_ref,
	}

	dc, _ := cli.ImageList(ctx, listOptions)

	imageID := strings.TrimPrefix(dc[0].ID, "sha256:")
	return imageID, nil
}

func imageExists(ctx context.Context, cli *client.Client, imageName string) (bool, error) {
	imageList, err := cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return false, err
	}

	for _, image := range imageList {
		for _, tag := range image.RepoTags {
			if tag == imageName {
				return true, nil
			}
		}
	}

	return false, nil
}
