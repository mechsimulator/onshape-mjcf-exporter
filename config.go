package main

import (
	"os"
	"encoding/json"
	"net/url"
	"strings"
	"errors"
)

type OnshapeClient struct {
	BaseUrl   string `json:"base_url"`
	SecretKey string `json:"secret_key"`
	AccessKey string `json:"access_key"`
}

type StlExportOptions struct {
	Units string `json:"units"`
	Mode string `json:"mode"`
}

type ExporterConfig struct {
	OnshapeClient OnshapeClient `json:"onshape_client"`
	StlExportOptions StlExportOptions `json:"stl_export_options"`
	BaseElement *OnshapeElement
}

func OnshapeElementFromURL(baseUrl string) *OnshapeElement {
	parseError := func(err error) {
		fatalError("failed to parse onshape url", err)
	}

	u, err := url.Parse(baseUrl)
	if err != nil {
		parseError(err)
	}

	segments := strings.Split(u.Path[1:], "/")
	if len(segments) != 6 {
		parseError(errors.New("malformed url"))
	}

	return &OnshapeElement{
		ServerURL: u.Scheme + "://" + u.Host + "/",
		did:       segments[1],
		wvm:       segments[2],
		wvmid:     segments[3],
		eid:       segments[5],
	}
}

func LoadConfigFromFile(path string) *ExporterConfig {
	contents, err := os.ReadFile(path)
	if err != nil {
		fatalError("failed to read onshape client config", err)
	}

	var configJson ExporterConfig
	err = json.Unmarshal(contents, &configJson)
	if err != nil {
		fatalError("failed to parse onshape client config", err)
	}

	configJson.BaseElement = OnshapeElementFromURL(configJson.OnshapeClient.BaseUrl)

	return &configJson
}