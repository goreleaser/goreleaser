package yaml

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestYAMLMarshalError(t *testing.T) {
	t.Parallel()

	type yamlMarshalError struct {
		BoolValue bool   `yaml:"bool_value"`
		FuncValue func() `yaml:"func_value"`
	}

	v := yamlMarshalError{
		BoolValue: true,
		FuncValue: func() {},
	}

	require.Panics(t, func() {
		_, _ = Marshal(v)
	})
}

func TestYAML(t *testing.T) {
	t.Parallel()

	type (
		yamlKey struct {
			BoolValue   bool    `yaml:"bool_value"`
			NumberValue float64 `yaml:"number_value"`
			StringValue string  `yaml:"string_value"`
		}
		yamlObject struct {
			Array []string `yaml:"array"`
			Key   yamlKey  `yaml:"key"`
		}
		yamlReceiver struct {
			Object yamlObject `yaml:"object"`
		}

		interfaceOrObject struct {
			obj   yamlReceiver
			iface interface{}
		}
	)

	for _, toPin := range []struct {
		Title        string
		InputYAML    []byte
		ExpectedYAML []byte            // optional: when marshalled YAML is expected to differ from raw input (e.g. on bool flags)
		Expected     interfaceOrObject // maybe either untyped interface{} or yamlObject struct with struct tags
		ExpectError  bool
		WantsStrict  bool // apply Strict mode
	}{
		{
			Title:     "happy path, untyped",
			InputYAML: testYAMLObject(),
			Expected: interfaceOrObject{
				iface: map[string]interface{}{
					"object": map[string]interface{}{
						"key": map[string]interface{}{
							"string_value": "This is a doc.\nOn multiple lines.\n",
							"bool_value":   "y",
							"number_value": 10.23,
						},
						"array": []interface{}{"x", "y"},
					},
				},
			},
		},
		{
			Title:       "happy path strict, untyped",
			InputYAML:   testYAMLObject(),
			WantsStrict: true,
			Expected: interfaceOrObject{
				iface: map[string]interface{}{
					"object": map[string]interface{}{
						"key": map[string]interface{}{
							"string_value": "This is a doc.\nOn multiple lines.\n",
							"bool_value":   "y",
							"number_value": 10.23,
						},
						"array": []interface{}{"x", "y"},
					},
				},
			},
		},
		{
			Title:        "happy path strict, explicit target",
			InputYAML:    testYAMLObject(),
			ExpectedYAML: testYAMLObjectBool(),
			WantsStrict:  true,
			Expected: interfaceOrObject{
				iface: nil,
				obj: yamlReceiver{
					Object: yamlObject{
						Key: yamlKey{
							StringValue: "This is a doc.\nOn multiple lines.\n",
							BoolValue:   true,
							NumberValue: 10.23,
						},
						Array: []string{"x", "y"},
					},
				},
			},
		},
		{
			Title:        "happy path non-strict, explicit target",
			InputYAML:    testYAMLObjectNonStrict(),
			ExpectedYAML: testYAMLObjectBool(),
			WantsStrict:  false,
			Expected: interfaceOrObject{
				iface: nil,
				obj: yamlReceiver{
					Object: yamlObject{
						Key: yamlKey{
							StringValue: "This is a doc.\nOn multiple lines.\n",
							BoolValue:   true,
							NumberValue: 10.23,
						},
						Array: []string{"x", "y"},
					},
				},
			},
		},
		{
			Title:        "happy path strict, explicit target: unknown field failure",
			InputYAML:    testYAMLObjectNonStrict(),
			ExpectedYAML: testYAMLObjectBool(),
			WantsStrict:  true,
			ExpectError:  true,
			Expected: interfaceOrObject{
				iface: nil,
				obj: yamlReceiver{
					Object: yamlObject{
						Key: yamlKey{
							StringValue: "This is a doc.\nOn multiple lines.\n",
							BoolValue:   true,
							NumberValue: 10.23,
						},
						Array: []string{"x", "y"},
					},
				},
			},
		},
	} {
		testCase := toPin

		t.Run(testCase.Title, func(t *testing.T) {
			t.Parallel()

			var (
				err               error
				b, expectedOutput []byte
			)
			iface := testCase.Expected.iface
			obj := testCase.Expected.obj

			expectedInput := toPlainYaml(testCase.InputYAML)

			if testCase.WantsStrict {
				if iface != nil {
					err = UnmarshalStrict(expectedInput, &iface)
				} else {
					err = UnmarshalStrict(expectedInput, &obj)
				}
			} else {
				if iface != nil {
					err = Unmarshal(expectedInput, &iface)
				} else {
					err = Unmarshal(expectedInput, &obj)
				}
			}

			if testCase.ExpectError {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)

			if iface != nil {
				require.EqualValues(t, testCase.Expected.iface, iface)

				b, err = Marshal(iface)
				require.NoError(t, err)
			} else {
				require.EqualValues(t, testCase.Expected.obj, obj)

				b, err = Marshal(obj)
				require.NoError(t, err)
			}

			if testCase.ExpectedYAML == nil {
				expectedOutput = expectedInput
			} else {
				expectedOutput = toPlainYaml(testCase.ExpectedYAML)
			}

			require.EqualValues(t, expectedOutput, b)
		})
	}
}

func toPlainYaml(in []byte) []byte {
	// ensure we got legit yaml for go strings that may have been reindented using tabs, or leading new CR in source.
	return bytes.ReplaceAll(
		bytes.TrimLeft(in, "\n\r"),
		[]byte("\t"), bytes.Repeat([]byte(" "), 4),
	)
}

func testYAMLObject() []byte {
	return []byte(`
object:
  array:
    - x
    - "y"
  key:
    bool_value: "y"
    number_value: 10.23
    string_value: |
      This is a doc.
      On multiple lines.
`)
}

func testYAMLObjectBool() []byte {
	// same object, but the "y" YAML for bool has been marshaled as "true"
	return []byte(`
object:
  array:
    - x
    - "y"
  key:
    bool_value: true
    number_value: 10.23
    string_value: |
      This is a doc.
      On multiple lines.
`)
}

func testYAMLObjectNonStrict() []byte {
	// same object, but with an extra unknown value
	return []byte(`
object:
  array:
    - x
    - "y"
  key:
    bool_value: true
    number_value: 10.23
    string_value: |
      This is a doc.
      On multiple lines.
  unknown: 'wrong'
`)
}
