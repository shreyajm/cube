package config

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type httpConfig struct {
	BaseConfig
	Port int `json:"port"`
}

func (h *httpConfig) UnmarshalJSON(b []byte) error {
	v := map[string]int{}
	if e := json.Unmarshal(b, &v); e != nil {
		return e
	}
	if p, ok := v["port"]; ok {
		h.Port = p
	} else {
		return fmt.Errorf("port must be present")
	}
	return nil
}

type loggerConfig struct {
	BaseConfig
	File string `json:"file"`
}

func TestJSONStore(t *testing.T) {
	Convey("On a json store", t, func() {
		goodJSON := strings.NewReader(`{"http": {"port": 8080}}
			{"logger": {"file": "/var/log/test.log"}}`)
		s := NewJSONStore(goodJSON)
		So(s, ShouldNotBeNil)
		defer s.Close()
		Convey("Should be able to load the file", func() {
			So(s.Open(), ShouldBeNil)
			Convey("should be able load http config", func() {
				cfg := &httpConfig{BaseConfig{"http"}, 0}
				err := s.Get(cfg)
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeNil)
				So(cfg.Port, ShouldEqual, 8080)
			})
			Convey("should not find randon config", func() {
				err := s.Get(&httpConfig{BaseConfig{"some_random_key"}, 0})
				So(err, ShouldBeError)
			})
			Convey("should return default config on empty key", func() {
				cfg := &httpConfig{BaseConfig{""}, 9999}
				So(s.Get(cfg), ShouldBeNil)
				So(cfg.Port, ShouldEqual, 9999)
			})
			Convey("should return nil on nil config", func() {
				So(s.Get(nil), ShouldBeNil)
			})
			Convey("should not be able to find wrong logger type", func() {
				err := s.Get(&httpConfig{BaseConfig{"logger"}, 0})
				So(err, ShouldBeError)
			})
			Convey("should be able to find logger", func() {
				cfg := &loggerConfig{BaseConfig{"logger"}, ""}
				err := s.Get(cfg)
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeNil)
				So(cfg.File, ShouldEqual, "/var/log/test.log")
			})
		})
	})
}

func TestBadJSON(t *testing.T) {
	Convey("Load bad json data", t, func() {
		Convey("should be a json parse error", func() {
			badJSON := strings.NewReader(`{"http": {"portx": "8080"}`)
			s := NewJSONStore(badJSON)
			So(s.Open(), ShouldBeError)
		})

		Convey("should error out on bad http config", func() {
			badKeyJSON := strings.NewReader(`{"http": {"portx": "8080"}}`)
			s := NewJSONStore(badKeyJSON)
			So(s.Open(), ShouldBeNil)
			So(s.Get(&httpConfig{BaseConfig{"http"}, 0}), ShouldBeError)
		})
	})
}
