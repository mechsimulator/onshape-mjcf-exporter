package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/onshape-public/go-client/onshape"
)

type OnshapeElement struct {
	did string
	wvm string
	wvmid string
	eid string
}

type OnshapeClientConfigJSON struct {
	BaseUrl string		`json:"base_url"`
	SecretKey string	`json:"secret_key"`
	AccessKey string	`json:"access_key"`
}

type OnshapeClientConfig struct {
	SecretKey string
	AccessKey string
	Element *OnshapeElement
}

func fatalError(message string, err error) {
	panic(fmt.Sprintf("%s: %s", message, err.Error()))
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
		did: segments[1],
		wvm: segments[2],
		wvmid: segments[3],
		eid: segments[5],
	}
}

func LoadConfigFromFile(path string) *OnshapeClientConfig {
	contents, err := os.ReadFile(path)
	if err != nil {
		fatalError("failed to read onshape client config", err)
	}

	var configJson OnshapeClientConfigJSON
	err = json.Unmarshal(contents, &configJson)
	if err != nil {
		fatalError("failed to parse onshape client config", err)
	}

	return &OnshapeClientConfig{
		SecretKey: configJson.SecretKey,
		AccessKey: configJson.AccessKey,
		Element: OnshapeElementFromURL(configJson.BaseUrl),
	}
}

func main() {
	config := onshape.NewConfiguration()
	config.Debug = true

	onshapeConfig := LoadConfigFromFile("./.onshape_client_config.json")
	e := onshapeConfig.Element

	client := onshape.NewAPIClient(config)

	ctx := context.WithValue(context.Background(), onshape.ContextAPIKeys, 
		onshape.APIKeys{onshapeConfig.SecretKey, onshapeConfig.AccessKey})

	client.AssembliesApi.GetAssemblyDefinition(ctx, e.did, e.wvm, e.wvmid, e.eid)
}