import re

with open("internal/pipe/gentoo/gentoo.go", "r") as f:
    content = f.read()

# Make NewExtraFileValidator and a method Filter(extraFiles map[string]string) error
insert_code = """
func NewExtraFileValidator(cfg config.Gentoo, arches []*artifact.Artifact) *extraFileValidator {
	return &extraFileValidator{
		cfg:    cfg,
		arches: arches,
	}
}

func (v *extraFileValidator) Filter(extraFiles map[string]string) error {
	for name, src := range extraFiles {
		if v.inArchives(name) {
			log.Warnf("file %s is already in all archives, skipping upload to Gentoo files/ directory", name)
			delete(extraFiles, name)
			continue
		}
		if err := v.validate(name, src); err != nil {
			return err
		}
	}
	return nil
}
"""

# Replace the instantiation and loop logic
search_str1 = """func (v *extraFileValidator) validate(name, src string) error {"""
if search_str1 in content:
    content = content.replace(search_str1, insert_code + search_str1)

search_str2 = """	validator := &extraFileValidator{cfg: cfg, arches: arches}"""
insert_code2 = """	validator := NewExtraFileValidator(cfg, arches)"""
if search_str2 in content:
    content = content.replace(search_str2, insert_code2)

search_str3 = """	for name, src := range extraFiles {
		if validator.inArchives(name) {
			log.Warnf("file %s is already in all archives, skipping upload to Gentoo files/ directory", name)
			delete(extraFiles, name)
			continue
		}
		if err := validator.validate(name, src); err != nil {
			return err
		}
	}"""
insert_code3 = """	if err := validator.Filter(extraFiles); err != nil {
		return err
	}"""

if search_str3 in content:
    content = content.replace(search_str3, insert_code3)

with open("internal/pipe/gentoo/gentoo.go", "w") as f:
    f.write(content)
