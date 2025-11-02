package config

import "github.com/invopop/jsonschema"

func (bh Hook) JSONSchema() *jsonschema.Schema {
	type hookAlias Hook
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	schema := reflector.Reflect(&hookAlias{})
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{
				Type: "string",
			},
			schema,
		},
	}
}

func (f File) JSONSchema() *jsonschema.Schema {
	type fileAlias File
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	schema := reflector.Reflect(&fileAlias{})
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{
				Type: "string",
			},
			schema,
		},
	}
}

func (bhc Hooks) JSONSchema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	var t Hook
	schema := reflector.Reflect(&t)
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{{
			Type: "string",
		}, {
			Type:  "array",
			Items: schema,
		}},
	}
}

func (a FlagArray) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{{
			Type: "string",
		}, {
			Type: "array",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		}},
	}
}

func (a StringArray) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{{
			Type: "string",
		}, {
			Type: "array",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		}},
	}
}

func (a NixDependency) JSONSchema() *jsonschema.Schema {
	type nixDependencyAlias NixDependency
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	schema := reflector.Reflect(&nixDependencyAlias{})
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{
				Type: "string",
			},
			schema,
		},
	}
}

func (a PullRequestBase) JSONSchema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	schema := reflector.Reflect(&pullRequestBase{})
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{
				Type: "string",
			},
			schema,
		},
	}
}

func (a HomebrewDependency) JSONSchema() *jsonschema.Schema {
	type homebrewDependencyAlias HomebrewDependency
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	schema := reflector.Reflect(&homebrewDependencyAlias{})
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{
				Type: "string",
			},
			schema,
		},
	}
}
