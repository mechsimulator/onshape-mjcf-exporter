package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
)

var StlUrlRedirect string

func CheckRedirectFunc(req *http.Request, via []*http.Request) error {
	// StlUrlRedirects = append(StlUrlRedirects, req.URL.String())
	StlUrlRedirect = req.URL.String()
	return http.ErrAbortHandler
}

func StlRequest(client *http.Client, ctx context.Context, url string, auth string) []byte {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		fatalError("failed to make stl request:", err)
	}
	req.Header.Add("accept", "application/octet-stream")
	req.Header.Add("Authorization", auth)
	resp, err := client.Do(req)
	if err != nil {
		fatalError("request failed:", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fatalError("failed to read stl response body:", err)
	}

	return body
}

func SaveStlsToDir(o *Onshape, parts []PartInfo, path string) {
	if err := os.MkdirAll(path, 0700); err != nil {
		fatalError(fmt.Sprint("failed to create stl directory at:", path), err)
	} 

	// Get STL redirect URLs because they are on a different server
	for _, part := range parts {
		_, resp, err := o.Client.PartApi.ExportStl(o.Ctx, part.Path.did, part.Path.wvm, part.Path.wvmid, part.Path.eid, part.Id).Mode(o.Config.StlExportOptions.Mode).Units(o.Config.StlExportOptions.Units).Execute()
		if err != nil || (resp != nil && resp.StatusCode >= 300) {
			fatalError("failed to request stl:", err)
		}

		clientConfig := o.Config.OnshapeClient
		stl := StlRequest(o.Client.GetConfig().HTTPClient, o.Ctx, StlUrlRedirect, MakeAuthorizationHeader(clientConfig.AccessKey, clientConfig.SecretKey))
		os.WriteFile(path + "/" + part.Name + ".stl", stl, 0700)
	}

	// Do actual STL fetching
	// for _, url := range StlUrlRedirects {
	// 	clientConfig := o.Config.OnshapeClient
	// 	stl := StlRequest(o.Client.GetConfig().HTTPClient, o.Ctx, url, MakeAuthorizationHeader(clientConfig.AccessKey, clientConfig.SecretKey))
	// 	os.WriteFile(path + "/" + )
	// }
}