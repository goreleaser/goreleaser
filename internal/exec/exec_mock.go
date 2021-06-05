package exec

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
)

// nolint: gochecknoglobals
var (
	MockEnvVar = "GORELEASER_MOCK_DATA"
	MockCmd    = os.Args[0]
)

type MockData struct {
	AnyOf []MockCall `json:"any_of,omitempty"`
}

type MockCall struct {
	Stdout       string   `json:"stdout,omitempty"`
	Stderr       string   `json:"stderr,omitempty"`
	ExpectedArgs []string `json:"args"`
	ExpectedEnv  []string `json:"env"`
	ExitCode     int      `json:"exit_code"`
}

func (m *MockData) MarshalJSON() ([]byte, error) {
	type t MockData
	return json.Marshal((*t)(m))
}

func (m *MockData) UnmarshalJSON(b []byte) error {
	type t MockData
	return json.Unmarshal(b, (*t)(m))
}

// MarshalMockEnv mocks marshal.
//
// nolint: interfacer
func MarshalMockEnv(data *MockData) string {
	b, err := data.MarshalJSON()
	if err != nil {
		errData := &MockData{
			AnyOf: []MockCall{
				{
					Stderr:   fmt.Sprintf("unable to marshal mock data: %s", err),
					ExitCode: 1,
				},
			},
		}
		b, _ = errData.MarshalJSON()
	}

	return MockEnvVar + "=" + string(b)
}

func ExecuteMockData(jsonData string) int {
	md := &MockData{}
	err := md.UnmarshalJSON([]byte(jsonData))
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to unmarshal mock data: %s", err)
		return 1
	}

	givenArgs := os.Args[1:]
	givenEnv := filterEnv(os.Environ())

	if len(md.AnyOf) == 0 {
		fmt.Fprintf(os.Stderr, "no mock calls expected. args: %q, env: %q",
			givenArgs, givenEnv)
		return 1
	}

	for _, item := range md.AnyOf {
		if item.ExpectedArgs == nil {
			item.ExpectedArgs = []string{}
		}
		if item.ExpectedEnv == nil {
			item.ExpectedEnv = []string{}
		}

		sort.Strings(givenEnv)
		sort.Strings(item.ExpectedEnv)
		sort.Strings(givenArgs)
		sort.Strings(item.ExpectedArgs)

		if reflect.DeepEqual(item.ExpectedArgs, givenArgs) &&
			reflect.DeepEqual(item.ExpectedEnv, givenEnv) {
			fmt.Fprint(os.Stdout, item.Stdout)
			fmt.Fprint(os.Stderr, item.Stderr)

			return item.ExitCode
		}
	}

	fmt.Fprintf(os.Stderr, "no mock calls matched. args: %q, env: %q",
		givenArgs, givenEnv)
	return 1
}

func filterEnv(vars []string) []string {
	for i, env := range vars {
		if strings.HasPrefix(env, MockEnvVar+"=") {
			return append(vars[:i], vars[i+1:]...)
		}
	}

	return vars
}
