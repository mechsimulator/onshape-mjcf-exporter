package main

import (
	"context"
	"encoding/base64"
	"fmt"

	"net/http"
	"github.com/onshape-public/go-client/onshape"
)

const BASE_MECHSIM_PATH = "C:\\Users\\Public\\MechSim"

type OnshapeElement struct {
	ServerURL string
	did       string
	wvm       string
	wvmid     string
	eid       string
}

type Onshape struct {
	Client  *onshape.APIClient
	Ctx     context.Context
	Config  *ExporterConfig
}

func MakeAuthorizationHeader(accessKey string, secretKey string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(accessKey + ":" + secretKey))
}

func NewOnshape(client *onshape.APIClient, ctx context.Context, config *ExporterConfig) Onshape {
	return Onshape{
		Client: client,
		Ctx:    ctx,
		Config: config,
	}
}

func checkRequest(response *http.Response, err error) bool {
	failed := err != nil || (response != nil && response.StatusCode >= 300)
	if failed {
		fmt.Println("err: ", err, " -- Response status: ", response)
	}
	return !failed
}

func fatalError(message string, err error) {
	panic(fmt.Sprintf("%s: %s", message, err.Error()))
}

const DEVMODE = true

func main() {
	config := onshape.NewConfiguration()
	config.Debug = true

	config.HTTPClient = &http.Client{
		CheckRedirect: CheckRedirectFunc,
	}

	var exporterConfig *ExporterConfig
	if DEVMODE {
		exporterConfig = LoadConfigFromFile("./tmp_config.json")
	} else {
		exporterConfig = LoadConfigFromFile("./.onshape_client_config.json")
	}

	client := onshape.NewAPIClient(config)
	ctx := context.WithValue(
		context.Background(),
		onshape.ContextBasicAuth,
		onshape.BasicAuth{
			UserName: exporterConfig.OnshapeClient.AccessKey,
			Password: exporterConfig.OnshapeClient.SecretKey,
		},
	)

	o := NewOnshape(client, ctx, exporterConfig)

	model := NewModelData(&o)
	for k, v := range model.Occurrences {
		fmt.Println(k, v)
	}
	
	

	// for _, url := range StlUrlRedirects {
	// 	StlRequest(client.GetConfig().HTTPClient, ctx, url, MakeAuthorizationHeader(onshapeConfig.AccessKey, onshapeConfig.SecretKey))
	// }

	// modelWriter := NewModelWriter(o)
	// modelWriter.MakeModel()
	// fmt.Println(modelWriter.ModelToString())
}
