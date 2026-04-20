package content

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

var fmDelim = []byte("---")

// splitFrontmatter separates optional YAML frontmatter from body.
// Returns (frontmatter, body). If no frontmatter, frontmatter is nil.
func splitFrontmatter(src []byte) ([]byte, []byte, error) {
	trimmed := bytes.TrimLeft(src, "\ufeff\r\n\t ")
	if !bytes.HasPrefix(trimmed, fmDelim) {
		return nil, src, nil
	}

	rest := trimmed[len(fmDelim):]
	if !bytes.HasPrefix(rest, []byte("\n")) && !bytes.HasPrefix(rest, []byte("\r\n")) {
		return nil, src, nil
	}

	end := bytes.Index(rest, append([]byte("\n"), fmDelim...))
	if end < 0 {
		return nil, nil, fmt.Errorf("unterminated frontmatter")
	}

	fm := rest[:end]
	body := rest[end+len(fmDelim)+1:]
	body = bytes.TrimLeft(body, "\r\n")

	return fm, body, nil
}

func parseFrontmatter(raw []byte) (Frontmatter, error) {
	var fm Frontmatter
	if len(raw) == 0 {
		return fm, nil
	}

	fm.ShowChildren = true

	if err := yaml.Unmarshal(raw, &fm); err != nil {
		return fm, fmt.Errorf("yaml: %w", err)
	}

	return fm, nil
}
