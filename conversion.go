package tinycast

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"pipelined.dev/audio/mp3"
	"pipelined.dev/pipe"
)

// BitRateMode describes how the MP3 encoder should apply bit rate limits.
type BitRateMode string

// BitRateModes are the modes accepted by the encoder.
var BitRateModes = []BitRateMode{
	"ABR",
	"CBR",
	"VBR",
}

// ParseBitRateMode returns a BitRateMode if it matches the given string or an
// error.
func ParseBitRateMode(in string) (BitRateMode, error) {
	for _, m := range BitRateModes {
		if in == string(m) {
			return m, nil
		}
	}
	return BitRateModes[0], fmt.Errorf("invalid bit rate mode '%s'", in)
}

// ToMp3BitRateMode converts a BitRateMode and BitRate to an mp3.BitRateMode.
func (brm BitRateMode) ToMp3BitRateMode(br BitRate) mp3.BitRateMode {
	switch brm {
	case BitRateModes[0]:
		return mp3.ABR(br)
	case BitRateModes[1]:
		return mp3.CBR(br)
	case BitRateModes[2]:
		return mp3.VBR(br)
	default:
		log.Panicf("Could not find bitrate")
		return mp3.ABR(0)
	}
}

// BitRate used to encode an audio file.
type BitRate int

// BitRates supported by the MP3 encoder.
var BitRates = []BitRate{
	16,
	32,
	64,
}

// ParseBitRate converts a string version of BitRate.
func ParseBitRate(in string) (BitRate, error) {
	for _, m := range BitRates {
		if in == m.ToString() {
			return m, nil
		}
	}
	return BitRates[0], fmt.Errorf("invalid bit rate '%s'", in)
}

// ToString returns a string version of the BitRate.
func (br BitRate) ToString() string {
	return strconv.FormatInt(int64(br), 10)
}

// ChannelModes supported by the MP3 encoder.
var ChannelModes = []mp3.ChannelMode{
	mp3.Mono,
	mp3.Stereo,
	mp3.JointStereo,
}

// ParseChannelMode translates from the string version of a ChannelMode.
func ParseChannelMode(in string) (mp3.ChannelMode, error) {
	for _, m := range ChannelModes {
		if in == m.String() {
			return m, nil
		}
	}
	return ChannelModes[0], fmt.Errorf("invalid channel mode '%s'", in)
}

func transform(c context.Context, cfg ConversionConfig, out io.Writer) error {
	resp, err := http.Get(cfg.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bufferSize := 1024 * 1024
	p, err := pipe.New(bufferSize, pipe.Line{
		Source: mp3.Source(resp.Body),
		Sink:   mp3.Sink(out, cfg.BitRateMode.ToMp3BitRateMode(cfg.BitRate), cfg.ChannelMode, mp3.DefaultEncodingQuality),
	})
	if err != nil {
		log.Fatalf("failed to bind line: %v", err)
	}
	err = pipe.Wait(p.Start(c))
	if err != nil {
		return err
	}
	return nil
}
