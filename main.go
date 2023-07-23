package main

import (
	"context"
	"encoding/json"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/onshape-public/go-client/onshape"
)

type OnshapeElement struct {
	ServerURL string
	did       string
	wvm       string
	wvmid     string
	eid       string
}

type OnshapeClientConfigJSON struct {
	BaseUrl   string `json:"base_url"`
	SecretKey string `json:"secret_key"`
	AccessKey string `json:"access_key"`
}

type OnshapeClientConfig struct {
	SecretKey string
	AccessKey string
	Element   *OnshapeElement
}

const (
	ASSEMBLY_DEF_REQ = "assemblyDefReq"
	DOCUMENT_REQ     = "documentReq"
)

type Response interface{}
type ResponseMap map[string]Response

type Onshape struct {
	Client  *onshape.APIClient
	Ctx     context.Context
	Element *OnshapeElement

	Responses ResponseMap
}

func MakeAuthorizationHeader(accessKey string, secretKey string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(accessKey + ":" + secretKey))
}

func NewOnshape(client *onshape.APIClient, ctx context.Context, element *OnshapeElement) Onshape {
	return Onshape{
		Client:    client,
		Ctx:       ctx,
		Element:   element,
		Responses: make(ResponseMap),
	}
}

func (o *Onshape) AddResponse(respKey string, resp Response) {
	o.Responses[respKey] = resp
}

func (o *Onshape) GetResponse(respKey string) Response {
	response := o.Responses[respKey]
	if response == nil {
		fmt.Println("response\"", respKey, "\" was nil")
	}
	return response
}

func checkRequest(response *http.Response, err error) bool {
	failed := err != nil || (response != nil && response.StatusCode >= 300)
	if failed {
		fmt.Print("err: ", err, " -- Response status: ", response)
	}
	return !failed
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
		ServerURL: u.Scheme + "://" + u.Host + "/",
		did:       segments[1],
		wvm:       segments[2],
		wvmid:     segments[3],
		eid:       segments[5],
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
		Element:   OnshapeElementFromURL(configJson.BaseUrl),
	}
}

const DEVMODE = true

func main() {
	config := onshape.NewConfiguration()
	config.Debug = true

	config.HTTPClient = &http.Client{
		CheckRedirect: CheckRedirectFunc,
	}

	var onshapeConfig *OnshapeClientConfig
	if DEVMODE {
		onshapeConfig = LoadConfigFromFile("./tmp_config.json")
	} else {
		onshapeConfig = LoadConfigFromFile("./.onshape_client_config.json")
	}

	e := onshapeConfig.Element

	client := onshape.NewAPIClient(config)

	ctx := context.WithValue(context.Background(), onshape.ContextBasicAuth, onshape.BasicAuth{UserName: onshapeConfig.AccessKey, Password: onshapeConfig.SecretKey})

	o := NewOnshape(client, ctx, e)

	assemblyDef, resp, err := o.Client.AssemblyApi.GetAssemblyDefinition(o.Ctx, e.did, e.wvm, e.wvmid, e.eid).IncludeMateConnectors(true).IncludeMateFeatures(true).ExcludeSuppressed(true).Execute()
	if checkRequest(resp, err) {
		o.AddResponse(ASSEMBLY_DEF_REQ, assemblyDef)
	}
	doc, resp, err := o.Client.DocumentApi.GetDocument(ctx, e.did).Execute()
	if checkRequest(resp, err) {
		o.AddResponse(DOCUMENT_REQ, doc)
	}

	fmt.Println(e.ServerURL)

	parts := make(map[string]string)
	for _, part := range assemblyDef.Parts {
		if !*part.IsStandardContent {
			stl, resp, err := o.Client.PartApi.ExportStl(ctx, *part.DocumentId, e.wvm, e.wvmid, *part.ElementId, *part.PartId).Units("meter").Mode("binary").Execute()
			if checkRequest(resp, err) {
				fmt.Println(stl)
			}
		}
	}
	fmt.Println(parts)

	for _, url := range StlUrlRedirects {
		StlRequest(client.GetConfig().HTTPClient, ctx, url, MakeAuthorizationHeader(onshapeConfig.AccessKey, onshapeConfig.SecretKey))
	}

	// modelWriter := NewModelWriter(o)
	// modelWriter.MakeModel()
	// fmt.Println(modelWriter.ModelToString())
}
