package tinycast

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	_ "net/http/pprof"

	"pipelined.dev/audio/mp3"
	"pipelined.dev/pipe"
)

type BitRateMode string

var BitRateModes = []BitRateMode{
	"ABR",
	"CBR",
	"VBR",
}

func ParseBitRateMode(in string) (BitRateMode, error) {
	for _, m := range BitRateModes {
		if in == string(m) {
			return m, nil
		}
	}
	return BitRateModes[0], fmt.Errorf("Invalid bit rate mode '%s'.", in)
}

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

type BitRate int

var BitRates = []BitRate{
	16,
	32,
	64,
}

func ParseBitRate(in string) (BitRate, error) {
	for _, m := range BitRates {
		if in == m.ToString() {
			return m, nil
		}
	}
	return BitRates[0], fmt.Errorf("Invalid bit rate '%s'.", in)
}

func (br BitRate) ToString() string {
	return strconv.FormatInt(int64(br), 10)
}

var ChannelModes = []mp3.ChannelMode{
	mp3.Mono,
	mp3.Stereo,
	mp3.JointStereo,
}

func ParseChannelMode(in string) (mp3.ChannelMode, error) {
	for _, m := range ChannelModes {
		if in == m.String() {
			return m, nil
		}
	}
	return ChannelModes[0], fmt.Errorf("Invalid channel mode '%s'.", in)
}

func transform(c context.Context, cfg ConversionConfig, out io.Writer) error {
	resp, err := http.Get(cfg.Url)
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
