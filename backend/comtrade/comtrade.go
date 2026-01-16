package comtrade

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func findFileCaseInsensitive(directory, base, ext string) (string, error) {
	ext = strings.TrimPrefix(ext, ".")
	candidate := filepath.Join(directory, base+"."+ext)
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", fmt.Errorf("read dir %s: %w", directory, err)
	}
	target := strings.ToLower(base + "." + ext)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.ToLower(e.Name()) == target {
			return filepath.Join(directory, e.Name()), nil
		}
	}
	return "", fmt.Errorf("file not found (case-insensitive): %s", candidate)
}

func ParseComtrade(cfgPath string, datPath string) (*Metadata, *ChannelData, error) {
	cfg, err := ParseComtradeCFGOnly(cfgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CFG file: %w", err)
	}

	dat, err := ParseComtradeWithMetadata(datPath, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse DAT file: %w", err)
	}

	return cfg, dat, nil
}

func ParseComtradeCFGOnly(cfgPath string) (*Metadata, error) {
	cfgFile, err := os.Open(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CFG file: %w", err)
	}
	defer cfgFile.Close()

	cfg, err := ParseCFGFile(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CFG file: %w", err)
	}

	return cfg, nil
}

func ParseComtradeWithMetadata(datPath string, meta *Metadata) (*ChannelData, error) {
	datFile, err := os.Open(datPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open DAT file: %w", err)
	}
	defer datFile.Close()

	dat, err := ParseDATFile(datFile, meta)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DAT file: %w", err)
	}

	return dat, nil
}
