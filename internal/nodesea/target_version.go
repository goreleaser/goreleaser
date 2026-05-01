package nodesea

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// minTargetConstraint is the floor for Node target versions a build-tool
// Node ≥ v25.5 (LIEF backed) can produce SEAs for. The V2 blob format
// emitted by `--build-sea` is only readable by these versions:
//   - v22.20.0+ (back-ported to the v22 LTS line)
//   - v24.6.0+
//   - v25.0.0+
//
// Older Node releases (v18, v20, v22.0–v22.19, v23, v24.0–v24.5) only
// understand the V1 blob format and will reject any binary built here
// at runtime.
//
//nolint:gochecknoglobals
var minTargetConstraint = mustNewConstraint(">=22.20.0-0,<23.0.0 || >=24.6.0-0,<25.0.0 || >=25.0.0-0")

func mustNewConstraint(raw string) *semver.Constraints {
	c, err := semver.NewConstraint(raw)
	if err != nil {
		panic(fmt.Sprintf("nodesea: invalid constraint %q: %v", raw, err))
	}
	return c
}

// ValidateTargetNodeVersion returns nil when version is a valid Node
// target for the `--build-sea` code path (i.e. it can read the V2 blob
// format). The returned error describes the supported floor when it
// rejects.
//
// version may be prefixed with "v"; both "v22.20.0" and "22.20.0" parse.
func ValidateTargetNodeVersion(version string) error {
	raw := strings.TrimPrefix(strings.TrimSpace(version), "v")
	if raw == "" {
		return errors.New("nodesea: empty target node version")
	}
	v, err := semver.StrictNewVersion(raw)
	if err != nil {
		return fmt.Errorf("nodesea: parse target node version %q: %w", version, err)
	}
	if !minTargetConstraint.Check(v) {
		return fmt.Errorf(
			"nodesea: target node %s is not supported by --build-sea; "+
				"floor is v22.20.0 / v24.6.0 / v25.0.0 (older releases only read the legacy V1 blob format)",
			version,
		)
	}
	return nil
}
