package comtrade

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

type AnalogChannelData struct {
	ChannelNumber int       `json:"channel"`
	RawData       []int32   `json:"rawData"`
	RawDataFloat  []float32 `json:"rawDataFloat"`
}

type DigitalChannelData struct {
	ChannelNumber int    `json:"channel"`
	RawData       []int8 `json:"rawData"`
}

type ChannelData struct {
	Timestamps      []int32             `json:"timestamps"`
	AnalogChannels  []AnalogChannelData  `json:"analogChannels"`
	DigitalChannels []DigitalChannelData `json:"digitalChannels"`
}

func newChannelData() *ChannelData {
	return &ChannelData{
		Timestamps:      []int32{},
		AnalogChannels:  []AnalogChannelData{},
		DigitalChannels: []DigitalChannelData{},
	}
}

func (dat *ChannelData) AddTimestampData(timestamp int32) {
	dat.Timestamps = append(dat.Timestamps, timestamp)
}

func (dat *ChannelData) AddAnalogData(channelNumber int, value int32) {
	for i := range dat.AnalogChannels {
		if dat.AnalogChannels[i].ChannelNumber == channelNumber {
			dat.AnalogChannels[i].RawData = append(dat.AnalogChannels[i].RawData, value)
			return
		}
	}
	dat.AnalogChannels = append(dat.AnalogChannels, AnalogChannelData{
		ChannelNumber: channelNumber,
		RawData:       []int32{value},
	})
}

func (dat *ChannelData) AddAnalogDataFloat(channelNumber int, value float32) {
	for i := range dat.AnalogChannels {
		if dat.AnalogChannels[i].ChannelNumber == channelNumber {
			dat.AnalogChannels[i].RawDataFloat = append(dat.AnalogChannels[i].RawDataFloat, value)
			return
		}
	}
	dat.AnalogChannels = append(dat.AnalogChannels, AnalogChannelData{
		ChannelNumber: channelNumber,
		RawDataFloat:  []float32{value},
	})
}

func (dat *ChannelData) AddDigitalData(channelNumber int, value int8) {
	for i := range dat.DigitalChannels {
		if dat.DigitalChannels[i].ChannelNumber == channelNumber {
			dat.DigitalChannels[i].RawData = append(dat.DigitalChannels[i].RawData, value)
			return
		}
	}
	dat.DigitalChannels = append(dat.DigitalChannels, DigitalChannelData{
		ChannelNumber: channelNumber,
		RawData:       []int8{value},
	})
}

func parseDATFile(f io.Reader, cfg *Metadata) (*ChannelData, error) {
	switch cfg.DataFileType {
	case "ascii":
		return parseDATFileASCII(f, cfg)
	case "binary", "binary32", "float32":
		return parseDATFileBinary(f, cfg)
	default:
		return nil, fmt.Errorf("unsupported data file type: %s", cfg.DataFileType)
	}
}

func parseDATFileASCII(f io.Reader, cfg *Metadata) (*ChannelData, error) {
	scanner := bufio.NewScanner(f)
	dat := newChannelData()

	na := cfg.AnalogChannelNum
	nd := cfg.DigitalChannelNum
	if na < 0 || nd < 0 {
		return nil, fmt.Errorf("invalid channel counts: NA=%d ND=%d", na, nd)
	}

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}

		// ASCII格式: n,timestamp,a1,a2,...,ana,d1,d2,...,dnd
		// 使用fmt.Sscanf或手动解析
		var n uint32
		var ts int32

		// 简化解析：使用fmt.Sscanf读取前两个值，然后手动解析其余
		parts := splitCommaLine(line)
		if len(parts) < 2+na+nd {
			return nil, fmt.Errorf("line %d: expected at least %d fields, got %d", lineNum, 2+na+nd, len(parts))
		}

		// 解析样本序号和时间戳
		if _, err := fmt.Sscanf(parts[0], "%d", &n); err != nil {
			return nil, fmt.Errorf("line %d: parse sample index: %w", lineNum, err)
		}
		if _, err := fmt.Sscanf(parts[1], "%d", &ts); err != nil {
			return nil, fmt.Errorf("line %d: parse timestamp: %w", lineNum, err)
		}
		dat.AddTimestampData(ts)

		// 解析模拟通道值(支持整数和浮点数)
		for i := range na {
			var val float64
			if _, err := fmt.Sscanf(parts[2+i], "%f", &val); err != nil {
				return nil, fmt.Errorf("line %d: parse analog ch %d: %w", lineNum, i+1, err)
			}
			dat.AddAnalogDataFloat(i+1, float32(val))
		}

		// 解析数字通道值
		for i := range nd {
			var val int
			if _, err := fmt.Sscanf(parts[2+na+i], "%d", &val); err != nil {
				return nil, fmt.Errorf("line %d: parse digital ch %d: %w", lineNum, i+1, err)
			}
			dat.AddDigitalData(i+1, int8(val))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return dat, nil
}

// splitCommaLine 按逗号分割行，处理空格
func splitCommaLine(line string) []string {
	parts := make([]string, 0)
	current := ""
	for _, c := range line {
		if c == ',' {
			parts = append(parts, current)
			current = ""
		} else if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

/*
| 数据 | 数据类型 | 字节数 |
| --- | --- | --- |
| 样本序号 (n) | 32位无符号整数 (unsigned long) | 4 字节 |
| 时间戳 (timestamp) | 32位无符号整数 (unsigned long) | 4 字节 |
| 模拟通道1 (A1) | 16位有符号整数 (short) | 2 字节 |
| 模拟通道2 (A2) | 16位有符号整数 (short) | 2 字节 |
| …（直到 ANA） |  |  |
| 数字通道 (D1...DND) | 16位无符号整数 (unsigned short) | 2 字节 |
*/
func parseDATFileBinary(f io.Reader, cfg *Metadata) (*ChannelData, error) {
	r := bufio.NewReader(f)
	dat := newChannelData()

	na := cfg.AnalogChannelNum
	nd := cfg.DigitalChannelNum
	if na < 0 || nd < 0 {
		return nil, fmt.Errorf("invalid channel counts: NA=%d ND=%d", na, nd)
	}
	digitalWords := (nd + 15) / 16

	for {
		// 样本序号 (n)
		var n uint32
		if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return nil, fmt.Errorf("read sample index: %w", err)
		}

		// 时间戳 (timestamp)
		var ts int32
		if err := binary.Read(r, binary.LittleEndian, &ts); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return nil, fmt.Errorf("read timestamp: %w", err)
		}
		dat.AddTimestampData(ts)

		// 模拟量 NA × int16
		for i := range na {
			switch cfg.DataFileType {
			case "binary":
				var raw int16
				if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
					if err == io.EOF || err == io.ErrUnexpectedEOF {
						return dat, nil
					}
					return nil, fmt.Errorf("read analog ch %d: %w", i+1, err)
				}
				dat.AddAnalogData(i+1, int32(raw))
			case "binary32":
				var raw int32
				if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
					if err == io.EOF || err == io.ErrUnexpectedEOF {
						return dat, nil
					}
					return nil, fmt.Errorf("read analog ch %d: %w", i+1, err)
				}
				dat.AddAnalogData(i+1, raw)
			case "float32":
				var raw float32
				if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
					if err == io.EOF || err == io.ErrUnexpectedEOF {
						return dat, nil
					}
					return nil, fmt.Errorf("read analog ch %d: %w", i+1, err)
				}
				dat.AddAnalogDataFloat(i+1, raw)
			default:
				return nil, fmt.Errorf("unsupported analog data type: %s", cfg.DataFileType)
			}
		}

		// 数字量打包字 ceil(ND/16) × uint16
		if digitalWords > 0 {
			packed := make([]uint16, digitalWords)
			for w := range digitalWords {
				if err := binary.Read(r, binary.LittleEndian, &packed[w]); err != nil {
					if err == io.EOF || err == io.ErrUnexpectedEOF {
						return dat, nil
					}
					return nil, fmt.Errorf("read digital word %d: %w", w, err)
				}
			}
			// 解包 ND 个数字量位
			for d := range nd {
				w := d / 16
				b := uint(d % 16)
				val := 0
				if ((packed[w] >> b) & 1) == 1 {
					val = 1
				}
				dat.AddDigitalData(d+1, int8(val))
			}
		}
	}

	return dat, nil
}

func ParseDATFile(f io.Reader, cfg *Metadata) (*ChannelData, error) {
	switch cfg.Version {
	case "1991", "1999", "2013":
		return parseDATFile(f, cfg)
	default:
		return nil, fmt.Errorf("unsupported COMTRADE version: %s", cfg.Version)
	}
}
