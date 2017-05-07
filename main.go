// ...coming soon...
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"

	"github.com/dustin/go-humanize"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

const filesClientSecret = "client_secret.json"
const filesToken = "token.json"

type bySize []*drive.File

func (items bySize) Len() int {
	return len(items)
}

func (items bySize) Swap(one int, two int) {
	items[one], items[two] = items[two], items[one]
}

func (items bySize) Less(one int, two int) bool {
	return items[one].QuotaBytesUsed > items[two].QuotaBytesUsed
}

func getClient(background context.Context, config *oauth2.Config) *http.Client {
	token := getToken(config)
	return config.Client(background, token)
}

func getConfig() *oauth2.Config {
	bytes, bytesErr := ioutil.ReadFile(filesClientSecret)
	if bytesErr != nil {
		log.Fatalf("%v\n", bytesErr)
	}

	config, configErr := google.ConfigFromJSON(bytes, drive.DriveMetadataReadonlyScope)
	if configErr != nil {
		log.Fatalf("%v\n", configErr)
	}

	return config
}

func getService(client *http.Client) *drive.Service {
	service, serviceErr := drive.New(client)
	if serviceErr != nil {
		log.Fatalf("%v\n", serviceErr)
	}

	return service
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

func fetch(pageSize int64) []*drive.File {
	files := []*drive.File{}
	background := context.Background()
	config := getConfig()
	client := getClient(background, config)
	service := getService(client)
	pageToken := ""
	for {
		query := service.
			Files.
			List().
			Fields("nextPageToken, files(id, name, quotaBytesUsed, webContentLink, webViewLink)").
			IncludeTeamDriveItems(true).
			OrderBy("quotaBytesUsed desc").
			PageSize(pageSize).
			Spaces("drive").
			SupportsTeamDrives(true)
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
	return files
}

func report(files []*drive.File, limit int) {
	log.Println("")

	totalFilesInt := len(files)
	totalFilesInt64 := int64(totalFilesInt)
	totalFilesHumanize := humanize.Comma(totalFilesInt64)
	log.Printf("Total Files: %v\n", totalFilesHumanize)

	totalBytesInt := 0
	totalBytesInt64 := int64(totalBytesInt)
	for _, file := range files {
		totalBytesInt64 = totalBytesInt64 + file.QuotaBytesUsed
	}
	totalBytesUint64 := uint64(totalBytesInt64)
	totalBytesHumanize := humanize.Bytes(totalBytesUint64)
	log.Printf("Total Size : %v\n", totalBytesHumanize)

	sort.Sort(bySize(files))

	if limit > totalFilesInt {
		limit = totalFilesInt
	}
	files = files[0:limit]

	log.Println("")

	for _, file := range files {
		bytesUint64 := uint64(file.QuotaBytesUsed)
		bytesHumanize := humanize.Bytes(bytesUint64)
		link := file.WebViewLink
		if link == "" {
			link = file.WebContentLink
		}
		log.Printf("%s [%9s] %s\n", link, bytesHumanize, file.Name)
	}
}

func main() {
	pageSize := flag.Int64("m", 999, "Page size")
	limit := flag.Int("l", 10, "Limit")
	flag.Parse()
	files := fetch(*pageSize)
	report(files, *limit)
}
