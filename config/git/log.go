package git

import "os/exec"

func Log(previous, current string) (str string, err error) {
	cmd := exec.Command(
		"git",
		"log",
		"--pretty=oneline",
		"--abbrev-commit",
		previous+".."+current,
	)
	bts, err := cmd.CombinedOutput()
	if err != nil {
		return str, err
	}
	return string(bts), err
}
