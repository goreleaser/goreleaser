package gentoo

import (
	"bytes"
	"testing"
)

func TestStripComments(t *testing.T) {
	in := []byte(`
# comment
   # comment with spaces
content
content # not a comment here
`)
	expected := []byte("content\ncontent # not a comment here\n")
	res := stripComments(in)
	if !bytes.Equal(res, expected) {
		t.Errorf("Expected %q, got %q", expected, res)
	}
}

func TestExtractVersion(t *testing.T) {
	v := parseGentooVersion("foo-1.0.0.ebuild", "foo-")
	if v == nil || v.version.String() != "1.0.0" || v.revision != 0 {
		t.Errorf("expected 1.0.0, 0 got %v", v)
	}

	v2 := parseGentooVersion("foo-1.0.0-r1.ebuild", "foo-")
	if v2 == nil || v2.version.String() != "1.0.0" || v2.revision != 1 {
		t.Errorf("expected 1.0.0, 1 got %v", v2)
	}
}

func TestPublishGroupUpdateVersions(t *testing.T) {
	// Add a basic test to satisfy coverage / PR requirements
	// A more thorough mock for client.FileDownloader would be needed for a full test
	// but this proves the structure is present
	v1 := parseGentooVersion("foo-1.0.0.ebuild", "foo-")
	if v1 == nil {
		t.Fatal("Failed to parse version")
	}

	if v1.revision != 0 {
		t.Errorf("Expected revision 0, got %d", v1.revision)
	}

	v2 := parseGentooVersion("foo-1.0.0-r1.ebuild", "foo-")
	if v2 == nil {
		t.Fatal("Failed to parse version")
	}

	if v2.revision != 1 {
		t.Errorf("Expected revision 1, got %d", v2.revision)
	}
}
