package comtrade

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

type Metadata struct {
	Station           string `json:"station"`
	Relay             string `json:"relay"`
	Version           string `json:"version"`
	TotalChannelNum   int    `json:"totalChannelNum"`
	AnalogChannelNum  int    `json:"analogChannelNum"`
	DigitalChannelNum int    `json:"digitalChannelNum"`
	AnalogChannels    []AnalogChannel `json:"analogChannels"`
	DigitalChannels   []DigitalChannel `json:"digitalChannels"`
	Frequency         float64 `json:"frequency"`
	RatesNum          int     `json:"ratesNum"`
	SampleRates       []SampleRate `json:"sampleRates"`
	StartTime         time.Time `json:"startTime"`
	EndTime           time.Time `json:"endTime"`
	DataFileType      string `json:"dataFileType"`
	TimeMultiplier    float64 `json:"timeMultiplier"`
}

type AnalogChannel struct {
	ChannelNumber int `json:"id"`
	ChannelName   string `json:"name"`
	Phase         string `json:"phase"`
	CCBM          string `json:"ccbm"`
	Unit          string `json:"unit"`
	Multiplier    float64 `json:"multiplier"`
	Offset        float64 `json:"offset"`
	Skew          float64 `json:"skew"`
	MinValue      float64 `json:"minValue"`
	MaxValue      float64 `json:"maxValue"`
	Primary       float64 `json:"primary"`
	Secondary     float64 `json:"secondary"`
	PS            string `json:"ps"`
}

type DigitalChannel struct {
	ChannelNumber int `json:"id"`
	ChannelName   string `json:"name"`
	Phase         string `json:"phase"`
	CCBM          string `json:"ccbm"`
	Y             int    `json:"y"`
}

type SampleRate struct {
	SampRate      float64 `json:"sampRate"`
	LastSampleNum int     `json:"lastSampleNum"`
}

func newMetadata() *Metadata {
	return &Metadata{
		AnalogChannels:  make([]AnalogChannel, 0),
		DigitalChannels: make([]DigitalChannel, 0),
		SampleRates:     make([]SampleRate, 0),
	}
}

type cfgParser struct {
	cfg     *Metadata
	status  int
	actions []func(*cfgParser, string) error
}

func newCFGParser() *cfgParser {
	return &cfgParser{
		cfg:    newMetadata(),
		status: 0,
		actions: []func(*cfgParser, string) error{
			parseStationLine,
		},
	}
}

func (p *cfgParser) parseLine(line string) error {
	if p.status >= len(p.actions) {
		return fmt.Errorf("unexpected extra line: %s", line)
	}
	if p.status == -1 {
		// parsing completed
		return nil
	}
	err := p.actions[p.status](p, line)
	if err != nil {
		return err
	}
	return nil
}

/*
first line: station_name,rec_dev_id,rev_year
example: STATION,RELAY,1999
*/
func parseStationLine(parser *cfgParser, line string) error {
	parts := splitAndTrim(line, ",")
	if len(parts) < 3 {
		return fmt.Errorf("invalid STATION line: %s", line)
	}
	parser.cfg.Station = parts[0]
	parser.cfg.Relay = parts[1]
	if len(parts) > 2 {
		parser.cfg.Version = parts[2]
	} else {
		parser.cfg.Version = "1991" // no version specified, assume 1991
	}
	switch parser.cfg.Version {
	case "1991", "1999", "2013":
		parser.status = 1
		parser.actions = append(parser.actions,
			parseChannelCountLine,
			parseAnalogChannelLine,
			parseDigitalChannelLine,
			parseFrequencyLine,
			parseRatesNumLine,
			parseSampleRateLine,
			parseStartTimeLine,
			parseEndTimeLine,
			parseDataFileTypeLine,
			parseTimeMultiplierLine,
		)
	default:
		return fmt.Errorf("unsupported COMTRADE version: %s", parser.cfg.Version)
	}
	return nil
}

/*
second line: TT,NA,ND
example: 6,3A,3D
*/
func parseChannelCountLine(parser *cfgParser, line string) error {
	parts := splitAndTrim(line, ",")
	if len(parts) < 3 {
		return fmt.Errorf("invalid CHANNEL COUNT line: %s", line)
	}
	var err error
	parser.cfg.TotalChannelNum, err = strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid TotalChannelNum: %s", parts[0])
	}
	parser.cfg.AnalogChannelNum, err = strconv.Atoi(strings.TrimSuffix(parts[1], "A"))
	if err != nil {
		return fmt.Errorf("invalid AnalogChannelNum: %s", parts[1])
	}
	parser.cfg.DigitalChannelNum, err = strconv.Atoi(strings.TrimSuffix(parts[2], "D"))
	if err != nil {
		return fmt.Errorf("invalid DigitalChannelNum: %s", parts[2])
	}
	parser.status++
	if parser.cfg.AnalogChannelNum == 0 {
		parser.status++
	}
	return nil
}

/*
analong channel line: An,ch_id,ph,ccbm,uu,a,b,skew,min,max,primary,secondary,PS
example: 1,保护电流A相,,,A,0.0043641975308642,0.0000000000000000,0,-32767,32767,1000.000000,1.000000,S
*/
func parseAnalogChannelLine(parser *cfgParser, line string) error {
	parts := splitAndTrim(line, ",")
	if len(parts) < 13 {
		return fmt.Errorf("invalid ANALOG CHANNEL line: %s", line)
	}
	var ch AnalogChannel
	var err error
	ch.ChannelNumber, err = strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid Analog ChannelNumber: %s", parts[0])
	}
	ch.ChannelName = parts[1]
	ch.Phase = parts[2]
	ch.CCBM = parts[3]
	ch.Unit = parts[4]
	ch.Multiplier, err = strconv.ParseFloat(parts[5], 64)
	if err != nil {
		return fmt.Errorf("invalid Analog Multiplier: %s", parts[5])
	}
	ch.Offset, err = strconv.ParseFloat(parts[6], 64)
	if err != nil {
		return fmt.Errorf("invalid Analog Offset: %s", parts[6])
	}
	ch.Skew, err = strconv.ParseFloat(parts[7], 64)
	if err != nil {
		return fmt.Errorf("invalid Analog Skew: %s", parts[7])
	}
	ch.MinValue, err = strconv.ParseFloat(parts[8], 64)
	if err != nil {
		return fmt.Errorf("invalid Analog MinValue: %s", parts[8])
	}
	ch.MaxValue, err = strconv.ParseFloat(parts[9], 64)
	if err != nil {
		return fmt.Errorf("invalid Analog MaxValue: %s", parts[9])
	}
	ch.Primary, err = strconv.ParseFloat(parts[10], 64)
	if err != nil {
		return fmt.Errorf("invalid Analog Primary: %s", parts[10])
	}
	ch.Secondary, err = strconv.ParseFloat(parts[11], 64)
	if err != nil {
		return fmt.Errorf("invalid Analog Secondary: %s", parts[11])
	}
	ch.PS = parts[12]

	parser.cfg.AnalogChannels = append(parser.cfg.AnalogChannels, ch)
	if len(parser.cfg.AnalogChannels) >= parser.cfg.AnalogChannelNum {
		parser.status++
		if parser.cfg.DigitalChannelNum == 0 {
			parser.status++
		}
	}
	return nil
}

/*
digital channel line: Dn,ch_id,ph,ccbm,y
example: 1,总启动,,,0
*/
func parseDigitalChannelLine(parser *cfgParser, line string) error {
	parts := splitAndTrim(line, ",")
	if len(parts) < 5 {
		return fmt.Errorf("invalid DIGITAL CHANNEL line: %s", line)
	}
	var ch DigitalChannel
	var err error
	ch.ChannelNumber, err = strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid Digital ChannelNumber: %s", parts[0])
	}
	ch.ChannelName = parts[1]
	ch.Phase = parts[2]
	ch.CCBM = parts[3]
	ch.Y, err = strconv.Atoi(parts[4])
	if err != nil {
		return fmt.Errorf("invalid Digital Y: %s", parts[4])
	}

	parser.cfg.DigitalChannels = append(parser.cfg.DigitalChannels, ch)
	if len(parser.cfg.DigitalChannels) >= parser.cfg.DigitalChannelNum {
		parser.status++
	}
	return nil
}

/*
frequency line: frequency
example: 50.000000
*/
func parseFrequencyLine(parser *cfgParser, line string) error {
	parts := splitAndTrim(line, ",")
	if len(parts) < 1 {
		return fmt.Errorf("invalid FREQUENCY line: %s", line)
	}
	freqFloat, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return fmt.Errorf("invalid Frequency: %s", parts[0])
	}
	parser.cfg.Frequency = freqFloat
	parser.status++
	return nil
}

/*
rates line: rates_num
example: 4
*/
func parseRatesNumLine(parser *cfgParser, line string) error {
	parts := splitAndTrim(line, ",")
	if len(parts) < 1 {
		return fmt.Errorf("invalid RATES NUM line: %s", line)
	}
	ratesNum, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid RatesNum: %s", parts[0])
	}
	parser.cfg.RatesNum = ratesNum
	parser.status++
	return nil
}

/*
sample rate line: samp_rate,last_sample_num
example: 4800.000000,24000
*/
func parseSampleRateLine(parser *cfgParser, line string) error {
	parts := splitAndTrim(line, ",")
	if len(parts) < 2 {
		return fmt.Errorf("invalid SAMPLE RATE line: %s", line)
	}
	var sr SampleRate
	var err error
	sr.SampRate, err = strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return fmt.Errorf("invalid SampRate: %s", parts[0])
	}
	sr.LastSampleNum, err = strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid LastSampleNum: %s", parts[1])
	}
	parser.cfg.SampleRates = append(parser.cfg.SampleRates, sr)
	if len(parser.cfg.SampleRates) >= parser.cfg.RatesNum {
		parser.status++
	}
	return nil
}

/*
start time line: start_time
example: 18/12/2023,16:32:06.351000
*/
func parseStartTimeLine(parser *cfgParser, line string) error {
	parts := splitAndTrim(line, ",")
	if len(parts) < 2 {
		return fmt.Errorf("invalid START TIME line: %s", line)
	}
	var err error
	parser.cfg.StartTime, err = time.Parse("02/01/2006,15:04:05.999999", line)
	if err != nil {
		return fmt.Errorf("invalid StartTime: %s", line)
	}
	parser.status++
	return nil
}

/*
end time line: end_time
example: 18/12/2023,16:32:11.351000
*/
func parseEndTimeLine(parser *cfgParser, line string) error {
	parts := splitAndTrim(line, ",")
	if len(parts) < 2 {
		return fmt.Errorf("invalid END TIME line: %s", line)
	}
	var err error
	parser.cfg.EndTime, err = time.Parse("02/01/2006,15:04:05.999999", line)
	if err != nil {
		return fmt.Errorf("invalid EndTime: %s", line)
	}
	parser.status++
	return nil
}

/*
data file type line: data_file_type
example: F
*/
func parseDataFileTypeLine(parser *cfgParser, line string) error {
	parts := splitAndTrim(line, ",")
	if len(parts) < 1 {
		return fmt.Errorf("invalid DATA FILE TYPE line: %s", line)
	}
	parser.cfg.DataFileType = strings.ToLower(parts[0])
	parser.status++
	return nil
}

/*
Time stamp multiplication factor: time_multiplier
example: 1.000000
*/
func parseTimeMultiplierLine(parser *cfgParser, line string) error {
	parts := splitAndTrim(line, ",")
	if len(parts) < 1 {
		return fmt.Errorf("invalid TIME MULTIPLIER line: %s", line)
	}
	var err error
	parser.cfg.TimeMultiplier, err = strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return fmt.Errorf("invalid TimeMultiplier: %s", parts[0])
	}
	parser.status = -1
	return nil
}

func ParseCFGFile(r io.Reader) (*Metadata, error) {
	encoding, err := detectEncoding(r)
	if err != nil {
		return nil, err
	}
	reader, err := transformToUTF8(r, encoding)
	if err != nil {
		return nil, err
	}

	// scan lines from decoded reader
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	parser := newCFGParser()
	for scanner.Scan() {
		line := scanner.Text()
		err := parser.parseLine(line)
		if err != nil {
			return nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return parser.cfg, nil
}
