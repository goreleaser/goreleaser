package config

import (
	"encoding/json"
	"strings"
)

// MarshalJSON marshals a slack block as JSON.
func (a SlackBlock) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Internal)
}

// UnmarshalYAML is a custom unmarshaler that unmarshals a YAML slack attachment as untyped interface{}.
func (a *SlackAttachment) UnmarshalYAML(unmarshal func(any) error) error {
	var yamlv2 any
	if err := unmarshal(&yamlv2); err != nil {
		return err
	}

	a.Internal = yamlv2

	return nil
}

// MarshalJSON marshals a slack attachment as JSON.
func (a SlackAttachment) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Internal)
}

// UnmarshalYAML is a custom unmarshaler that allows simplified declarations of commands as strings.
func (bh *Hook) UnmarshalYAML(unmarshal func(any) error) error {
	var cmd string
	if err := unmarshal(&cmd); err != nil {
		type t Hook
		var hook t
		if err := unmarshal(&hook); err != nil {
			return err
		}
		*bh = (Hook)(hook)
		return nil
	}

	bh.Cmd = cmd
	return nil
}

// UnmarshalYAML is a custom unmarshaler that wraps strings in arrays.
func (f *File) UnmarshalYAML(unmarshal func(any) error) error {
	type t File
	var str string
	if err := unmarshal(&str); err == nil {
		*f = File{Source: str}
		return nil
	}

	var file t
	if err := unmarshal(&file); err != nil {
		return err
	}
	*f = File(file)
	return nil
}

// UnmarshalYAML is a custom unmarshaler that allows simplified declaration of single command.
func (bhc *Hooks) UnmarshalYAML(unmarshal func(any) error) error {
	var singleCmd string
	err := unmarshal(&singleCmd)
	if err == nil {
		*bhc = []Hook{{Cmd: singleCmd}}
		return nil
	}

	type t Hooks
	var hooks t
	if err := unmarshal(&hooks); err != nil {
		return err
	}
	*bhc = (Hooks)(hooks)
	return nil
}

// UnmarshalYAML is a custom unmarshaler that wraps strings in arrays.
func (a *FlagArray) UnmarshalYAML(unmarshal func(any) error) error {
	var flags []string
	if err := unmarshal(&flags); err != nil {
		var flagstr string
		if err := unmarshal(&flagstr); err != nil {
			return err
		}
		*a = strings.Fields(flagstr)
	} else {
		*a = flags
	}
	return nil
}

// UnmarshalYAML is a custom unmarshaler that unmarshals a YAML slack block as untyped interface{}.
func (a *SlackBlock) UnmarshalYAML(unmarshal func(any) error) error {
	var yamlv2 any
	if err := unmarshal(&yamlv2); err != nil {
		return err
	}

	a.Internal = yamlv2

	return nil
}

// UnmarshalYAML is a custom unmarshaler that wraps strings in arrays.
func (a *StringArray) UnmarshalYAML(unmarshal func(any) error) error {
	var strings []string
	if err := unmarshal(&strings); err != nil {
		var str string
		if err := unmarshal(&str); err != nil {
			return err
		}
		*a = []string{str}
	} else {
		*a = strings
	}
	return nil
}

func (a *NixDependency) UnmarshalYAML(unmarshal func(any) error) error {
	var str string
	if err := unmarshal(&str); err == nil {
		a.Name = str
		return nil
	}

	type t NixDependency
	var dep t
	if err := unmarshal(&dep); err != nil {
		return err
	}

	a.Name = dep.Name
	a.OS = dep.OS

	return nil
}

// UnmarshalYAML is a custom unmarshaler that accept brew deps in both the old and new format.
func (a *PullRequestBase) UnmarshalYAML(unmarshal func(any) error) error {
	var str string
	if err := unmarshal(&str); err == nil {
		a.Branch = str
		return nil
	}

	var base pullRequestBase
	if err := unmarshal(&base); err != nil {
		return err
	}

	a.Branch = base.Branch
	a.Owner = base.Owner
	a.Name = base.Name

	return nil
}

// UnmarshalYAML is a custom unmarshaler that accept brew deps in both the old and new format.
func (a *HomebrewDependency) UnmarshalYAML(unmarshal func(any) error) error {
	var str string
	if err := unmarshal(&str); err == nil {
		a.Name = str
		return nil
	}

	type t HomebrewDependency
	var dep t
	if err := unmarshal(&dep); err != nil {
		return err
	}

	a.Name = dep.Name
	a.Type = dep.Type
	a.Version = dep.Version
	a.OS = dep.OS

	return nil
}
