package comtrade

import (
	"bytes"
	"fmt"
)

// ParseComtradeFromBytes 从字节数据解析COMTRADE文件
func ParseComtradeFromBytes(cfgData []byte, datData []byte) (*Metadata, *ChannelData, error) {
	cfg, err := ParseComtradeCFGFromBytes(cfgData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CFG data: %w", err)
	}

	dat, err := ParseComtradeWithMetadataFromBytes(datData, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse DAT data: %w", err)
	}

	return cfg, dat, nil
}

// ParseComtradeCFGFromBytes 从字节数据解析CFG
func ParseComtradeCFGFromBytes(cfgData []byte) (*Metadata, error) {
	reader := bytes.NewReader(cfgData)
	cfg, err := ParseCFGFile(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CFG data: %w", err)
	}

	return cfg, nil
}

// ParseComtradeWithMetadataFromBytes 从字节数据解析DAT
func ParseComtradeWithMetadataFromBytes(datData []byte, meta *Metadata) (*ChannelData, error) {
	reader := bytes.NewReader(datData)
	dat, err := ParseDATFile(reader, meta)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DAT data: %w", err)
	}

	return dat, nil
}
