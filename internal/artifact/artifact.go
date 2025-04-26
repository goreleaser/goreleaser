func cleanName(a Artifact) string {
	name := a.Name
	ext := filepath.Ext(name)
	result := strings.TrimSpace(strings.TrimSuffix(name, ext)) + ext
	if name != result {
		log.WithField("name", a.Name).
			WithField("new name", result).
			WithField("type", a.Type).
			WithField("path", a.Path).
			Warn("removed trailing whitespaces from artifact name")
	}
	return result
}
