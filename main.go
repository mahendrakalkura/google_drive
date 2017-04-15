// ...coming soon...
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

const filesClientSecret = "client_secret.json"
const filesToken = "token.json"

func getClient(background context.Context, config *oauth2.Config) *http.Client {
	token := getToken(config)
	return config.Client(background, token)
}

func getToken(config *oauth2.Config) *oauth2.Token {
	token, err := getTokenFromFile()
	if err == nil {
		return token
	}

	token = getTokenFromGoogle(config)
	setToken(token)
	return token
}

func getTokenFromFile() (*oauth2.Token, error) {
	file, fileErr := os.Open(filesToken)
	if fileErr != nil {
		return nil, fileErr
	}
	defer file.Close()

	token := &oauth2.Token{}
	err := json.NewDecoder(file).Decode(token)
	return token, err
}

func getTokenFromGoogle(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Println("Open the following URL in your browser:")
	fmt.Println(authURL)
	fmt.Printf("Code:")
	fmt.Printf(" ")

	code := ""
	_, scanErr := fmt.Scan(&code)
	if scanErr != nil {
		panic(scanErr)
	}

	token, tokenErr := config.Exchange(oauth2.NoContext, code)
	if tokenErr != nil {
		panic(tokenErr)
	}
	return token
}

func setToken(token *oauth2.Token) {
	file, fileErr := os.OpenFile(filesToken, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if fileErr != nil {
		panic(fileErr)
	}
	defer file.Close()
	json.NewEncoder(file).Encode(token)
}

func main() {
	background := context.Background()
	bytes, bytesErr := ioutil.ReadFile(filesClientSecret)
	if bytesErr != nil {
		panic(bytesErr)
	}

	config, configErr := google.ConfigFromJSON(bytes, drive.DriveMetadataReadonlyScope)
	if configErr != nil {
		panic(configErr)
	}

	client := getClient(background, config)

	server, serverErr := drive.New(client)
	if serverErr != nil {
		panic(serverErr)
	}

	files := []*drive.File{}

	pageToken := ""
	for {
		query := server.Files.
			List().
			Fields("nextPageToken, files(id, quotaBytesUsed, name)").
			OrderBy("name").
			PageSize(1000).
			Q("parents in 'root'")
		if pageToken != "" {
			query = query.PageToken(pageToken)
		}
		result, resultErr := query.Do()
		if resultErr != nil {
			panic(resultErr)
		}
		files = append(files, result.Files...)
		pageToken = result.NextPageToken
		if pageToken == "" {
			break
		}
	}

	len := len(files)
	if len > 0 {
		for _, file := range files {
			fmt.Println(fmt.Sprintf("%72s: [%9d] %s", file.Id, file.QuotaBytesUsed, file.Name))
		}
	}
}
