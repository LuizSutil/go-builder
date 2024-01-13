// A comment on the package will be output when you list the targets of a
// magefile.
package dc_emulator

import (
	"fmt"
	"os"
)

func PWD(volPath string) string {
	var path, _ = os.Getwd()
	currPath := fmt.Sprintf("%s/%s", path, volPath)
	return currPath
}

var ecs_deploy = map[string]dockerRunArgs{
	"deploy": {
		build:    ".",
		image:    "docker-test-python",
		env_file: []string{".env"},
		volumes:  []string{PWD("src:/work")},
		command:  []string{"/work/test.py"},
	},
}

// A comment on the package will be output when you list the targets of a
// magefile.
func EcsDeploy(cmd string) {

	cmdArgs := ecs_deploy[cmd]
	DockerApi(cmdArgs)

}
