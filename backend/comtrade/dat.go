package comtrade

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

type AnalogData struct {
	Value int16
}

type DigitalData struct {
	Value int8
}

type TimestampData struct {
	Timestamp uint32
}

type AnalogChannelData struct {
	ChannelNumber int
	Data          []AnalogData
}

type DigitalChannelData struct {
	ChannelNumber int
	Data          []DigitalData
}

type DAT struct {
	TimestampDatas      []TimestampData
	AnalogChannelDatas  []AnalogChannelData
	DigitalChannelDatas []DigitalChannelData
}

func newDAT() *DAT {
	return &DAT{
		TimestampDatas:      []TimestampData{},
		AnalogChannelDatas:  []AnalogChannelData{},
		DigitalChannelDatas: []DigitalChannelData{},
	}
}

func (dat *DAT) AddTimestampData(timestamp uint32) {
	dat.TimestampDatas = append(dat.TimestampDatas, TimestampData{
		Timestamp: timestamp,
	})
}

func (dat *DAT) AddAnalogData(channelNumber int, value int16) {
	for i := range dat.AnalogChannelDatas {
		if dat.AnalogChannelDatas[i].ChannelNumber == channelNumber {
			dat.AnalogChannelDatas[i].Data = append(dat.AnalogChannelDatas[i].Data, AnalogData{
				Value: value,
			})
			return
		}
	}
	dat.AnalogChannelDatas = append(dat.AnalogChannelDatas, AnalogChannelData{
		ChannelNumber: channelNumber,
		Data: []AnalogData{
			{
				Value: value,
			},
		},
	})
}

func (dat *DAT) AddDigitalData(channelNumber int, value int8) {
	for i := range dat.DigitalChannelDatas {
		if dat.DigitalChannelDatas[i].ChannelNumber == channelNumber {
			dat.DigitalChannelDatas[i].Data = append(dat.DigitalChannelDatas[i].Data, DigitalData{
				Value: value,
			})
			return
		}
	}
	dat.DigitalChannelDatas = append(dat.DigitalChannelDatas, DigitalChannelData{
		ChannelNumber: channelNumber,
		Data: []DigitalData{
			{
				Value: value,
			},
		},
	})
}

func parseDATFile1999(f io.Reader, cfg *Metadata) (*DAT, error) {
	switch cfg.DataFileType {
	case "ASCII":
		return parseDATFileASCII(f, cfg)
	case "BINARY":
		return parseDATFileBinary1999(f, cfg)
	default:
		return nil, fmt.Errorf("unsupported data file type: %s", cfg.DataFileType)
	}
}

func parseDATFileASCII(f io.Reader, cfg *Metadata) (*DAT, error) {
	return nil, nil
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
func parseDATFileBinary1999(f io.Reader, cfg *Metadata) (*DAT, error) {
	r := bufio.NewReader(f)
	dat := newDAT()

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
		var ts uint32
		if err := binary.Read(r, binary.LittleEndian, &ts); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return nil, fmt.Errorf("read timestamp: %w", err)
		}
		dat.AddTimestampData(ts)

		// 模拟量 NA × int16
		for i := range na {
			var raw int16
			if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					return dat, nil
				}
				return nil, fmt.Errorf("read analog ch %d: %w", i+1, err)
			}
			dat.AddAnalogData(i+1, raw)
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

func ParseDATFile(f io.Reader, cfg *Metadata) (*DAT, error) {
	switch cfg.Version {
	case "1999":
		return parseDATFile1999(f, cfg)
	// case "2013":
	// 	return parseDATFile2013(f, cfg)
	default:
		return nil, fmt.Errorf("unsupported COMTRADE version: %s", cfg.Version)
	}
}
