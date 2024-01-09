package dc_emulator

import "fmt"

var defaultImage string = "public.ecr.aws/dnxsolutions/ecs-deploy:latest"
var defaultVolume string = "/Users/luizsutil/Development/go-buider/go-src/src:/work"

var ecs_deploy = map[string]dockerRunArgs{
	"deploy": {
		build:    ".",
		image:    defaultImage,
		env_file: []string{".env"},
		volumes:  []string{defaultVolume},
		command:  []string{"/work/test.py"},
	},
}

func EcsDeploy(cmd string) {

	cmdArgs := ecs_deploy[cmd]
	containerName := fmt.Sprintf("%s-ecs", cmd)
	DockerApi(containerName, cmdArgs)

}
