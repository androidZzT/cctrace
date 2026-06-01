package collectors

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/androidZzT/cctrace/internal/trace"
)

type skillMetadata struct {
	Name        string
	Description string
	Path        string
}

func CollectSkills(sessionID string, roots []string) ([]trace.Event, error) {
	var skills []skillMetadata
	for _, root := range roots {
		if root == "" {
			continue
		}
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
				return nil
			}
			skill, err := readSkillMetadata(path)
			if err != nil || skill.Name == "" || skill.Description == "" {
				return nil
			}
			skills = append(skills, skill)
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}

	now := time.Now()
	events := make([]trace.Event, 0, len(skills))
	for _, skill := range skills {
		events = append(events, trace.Event{
			ID:             stableID("skill_startup", skill.Name, skill.Path),
			SessionID:      sessionID,
			Type:           trace.EventSkill,
			Title:          skill.Name,
			Timestamp:      now,
			Status:         trace.StatusOK,
			Source:         trace.SourceDerived,
			CorrelationIDs: []string{skill.Name},
			Confidence:     trace.ConfidenceExact,
			Summary:        map[string]any{"description": skill.Description, "path": skill.Path, "phase": "startup"},
			RawRef:         &trace.RawRef{File: skill.Path},
		})
	}
	return events, nil
}

func DefaultSkillRoots(home string) []string {
	return []string{filepath.Join(home, ".claude", "plugins", "cache")}
}

func readSkillMetadata(path string) (skillMetadata, error) {
	file, err := os.Open(path)
	if err != nil {
		return skillMetadata{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return skillMetadata{}, scanner.Err()
	}
	skill := skillMetadata{Path: path}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "---" {
			return skill, nil
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		switch strings.TrimSpace(key) {
		case "name":
			skill.Name = value
		case "description":
			skill.Description = value
		}
	}
	if err := scanner.Err(); err != nil {
		return skillMetadata{}, err
	}
	return skillMetadata{}, fmt.Errorf("missing frontmatter terminator in %s", path)
}
