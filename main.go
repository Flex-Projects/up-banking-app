package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/drive/v3"
)

//Use Service account
func ServiceAccount(secretFile string) *http.Client {
	b, err := ioutil.ReadFile(secretFile)
	if err != nil {
		log.Fatal("error while reading the credential file", err)
	}
	var s = struct {
		Email      string `json:"client_email"`
		PrivateKey string `json:"private_key"`
	}{}
	json.Unmarshal(b, &s)
	config := &jwt.Config{
		Email:      s.Email,
		PrivateKey: []byte(s.PrivateKey),
		Scopes: []string{
			drive.DriveScope,
		},
		TokenURL: google.JWTTokenURL,
	}
	client := config.Client(context.Background())
	return client
}

func createFolder(service *drive.Service, name string, parentId string) (*drive.File, error) {
	d := &drive.File{
		Name:     name,
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{parentId},
	}

	file, err := service.Files.Create(d).Do()

	if err != nil {
		log.Println("Could not create dir: " + err.Error())
		return nil, err
	}

	return file, nil
}

func createFile(service *drive.Service, name string, mimeType string, content io.Reader, parentId string) (*drive.File, error) {
	f := &drive.File{}
	file, err := service.Files.Update("1_ivOcqOG-MqS6BRJjh6bgl-cNmvvfv5U", f).Media(content).Do()

	if err != nil {
		log.Println("Could not create file: " + err.Error())
		return nil, err
	}

	return file, nil
}

func UploadFile(Filename string, Folder string) string {
	// Step 1: Open  file
	f, err := os.Open(Filename)

	if err != nil {
		panic(fmt.Sprintf("cannot open file: %v", err))
	}

	defer f.Close()

	// Step 2: Get the Google Drive service
	client := ServiceAccount("client_secret.json")

	srv, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve drive Client %v", err)
	}

	folderId := Folder

	// Step 4: create the file and upload
	file, err := createFile(srv, f.Name(), "application/octet-stream", f, folderId)

	if err != nil {
		panic(fmt.Sprintf("Could not create file: %v\n", err))
	}

	fmt.Printf("File '%s' successfully uploaded", file.Name)
	fmt.Printf("\nFile Id: '%s' ", file.Id)

	return file.Id
}

type TransactionValue struct {
	Value int `json:"valueInBaseUnits"`
}

type TransactionData struct {
	Data TransactionInnerData `json:"data"`
}

type TransactionInnerData struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

type TransactionAttr struct {
	Status      string           `json:"status"`
	Description string           `json:"description"`
	Message     string           `json:"message"`
	Amount      TransactionValue `json:"amount"`
	SettledDate string           `json:"settledAt"`
	CreatedDate string           `json:"createdAt"`
}

type TransactionRel struct {
	Category       TransactionData `json:"category"`
	ParentCategory TransactionData `json:"parentCategory"`
}

type Transaction struct {
	Type          string          `json:"type"`
	Id            string          `json:"id"`
	Attributes    TransactionAttr `json:"attributes"`
	Relationships TransactionRel  `json:"relationships"`
}

type TransactionLinks struct {
	Link string `json:"next"`
}
type Transactions struct {
	Data  []Transaction    `json:"data"`
	Links TransactionLinks `json:"links"`
}

func main() {
	client := &http.Client{
		CheckRedirect: http.DefaultClient.CheckRedirect,
	}

	mainTransactions := Transactions{}
	currentLink := ""
	done := false

	for !done {
		if currentLink == "" {
			fmt.Println("Starting!")
			currentLink = "https://api.up.com.au/api/v1/accounts/b36b1ce3-1444-4c9c-92b4-c527839e3ea1/transactions?filter[page]=100"
		} else {
			fmt.Println("Using link " + currentLink)
		}
		req, _ := http.NewRequest("GET", currentLink, nil)
		req.Header.Add("Authorization", `Bearer up:yeah:0YLVPc93cixD2ElxmAEtKBJYqaNvyTFrIBoyjuZWBy4c0EZZ8EWIyulytLGw6LiWM0XD8hQJZVoP1sOtzPjs2373DqLfQCVVofeUKAifZlBi5KIXafbHNmNdW1PPuoh5`)
		resp, err := client.Do(req)

		if err != nil {
			fmt.Print(err.Error())
			os.Exit(1)
		}

		responseData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		trans := &Transactions{}
		if err := json.Unmarshal(responseData, trans); err != nil {
			fmt.Printf("error decoding JSON: %v\n", err)
		}

		if trans.Links.Link == "" {
			done = true
		} else {
			currentLink = trans.Links.Link
		}
		mainTransactions.Data = append(mainTransactions.Data, trans.Data...)
	}

	csvFile, err := os.Create("./data.csv")

	if err != nil {
		fmt.Println(err)
	}

	defer csvFile.Close()
	writer := csv.NewWriter(csvFile)

	// i <3 aot
	for i := 0; i < len(mainTransactions.Data); i++ {
		var current = mainTransactions.Data[i]

		var row []string

		settleDate, _ := time.Parse(time.RFC3339, mainTransactions.Data[i].Attributes.SettledDate)
		createDate, _ := time.Parse(time.RFC3339, mainTransactions.Data[i].Attributes.CreatedDate)

		row = append(row, current.Type)
		row = append(row, current.Id)
		row = append(row, current.Attributes.Status)
		row = append(row, current.Attributes.Description)
		row = append(row, current.Attributes.Message)
		if len(current.Relationships.ParentCategory.Data.Id) == 0 {
			current.Relationships.ParentCategory.Data.Id = "Uncategorised"
		}
		row = append(row, current.Relationships.ParentCategory.Data.Id)
		if len(current.Relationships.Category.Data.Id) == 0 {
			current.Relationships.Category.Data.Id = "Uncategorised"
		}
		row = append(row, current.Relationships.Category.Data.Id)
		row = append(row, fmt.Sprint(current.Attributes.Amount.Value))
		row = append(row, settleDate.Local().Format("02-01-2006"))
		row = append(row, createDate.Local().Format("02-01-2006"))
		writer.Write(row)
	}

	UploadFile("data.csv", "1ZfVxRvifLAohKsTIVUeXtG0_2zdM9TYB")
}
