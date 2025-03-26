package exec

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
)

//nolint:gochecknoglobals
var (
	mockEnvVar = "GORELEASER_MOCK_DATA"
	mockCmd    = os.Args[0]
)

type mockData struct {
	AnyOf []mockCall `json:"any_of,omitempty"`
}

type mockCall struct {
	Stdout       string   `json:"stdout,omitempty"`
	Stderr       string   `json:"stderr,omitempty"`
	ExpectedArgs []string `json:"args"`
	ExpectedEnv  []string `json:"env"`
	ExitCode     int      `json:"exit_code"`
}

// MarshalJSON implements json.Marshaler.
func (m mockData) MarshalJSON() ([]byte, error) {
	type t mockData
	return json.Marshal((t)(m))
}

// UnmarshalJSON implements json.Unmarshaler.
func (m *mockData) UnmarshalJSON(b []byte) error {
	type t mockData
	return json.Unmarshal(b, (*t)(m))
}

// MarshalMockEnv mocks marshal.
//
//nolint:interfacer
func MarshalMockEnv(data *mockData) string {
	b, err := data.MarshalJSON()
	if err != nil {
		errData := &mockData{
			AnyOf: []mockCall{
				{
					Stderr:   fmt.Sprintf("unable to marshal mock data: %s", err),
					ExitCode: 1,
				},
			},
		}
		b, _ = errData.MarshalJSON()
	}

	return mockEnvVar + "=" + string(b)
}

// ExecuteMockData executes the mock data.
func ExecuteMockData(jsonData string) int {
	md := &mockData{}
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

		slices.Sort(givenEnv)
		slices.Sort(item.ExpectedEnv)
		slices.Sort(givenArgs)
		slices.Sort(item.ExpectedArgs)

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
		if strings.HasPrefix(env, mockEnvVar+"=") {
			return slices.Delete(vars, i, i+1)
		}
	}

	return vars
}
