package drift

import (
	"os"
	"path/filepath"
	"strings"
)

func displayPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Clean(path)
	}
	return homePathForDisplay(path, home)
}

func homePathForDisplay(path string, home string) string {
	cleanPath := filepath.Clean(path)
	cleanHome := filepath.Clean(home)
	if cleanHome == "." || cleanHome == string(filepath.Separator) {
		return cleanPath
	}
	if cleanPath == cleanHome {
		return "~"
	}
	prefix := cleanHome + string(filepath.Separator)
	if strings.HasPrefix(cleanPath, prefix) {
		return "~" + string(filepath.Separator) + strings.TrimPrefix(cleanPath, prefix)
	}
	return cleanPath
}

func layerDisplayPath(baseDir string, layer string) string {
	if rel, err := filepath.Rel(baseDir, layer); err == nil {
		if rel == "." {
			return filepath.Base(layer)
		}
		if !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
			return rel
		}
	}
	return displayPath(layer)
}
