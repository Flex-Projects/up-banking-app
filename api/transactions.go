package api

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func main() {
	client := &http.Client{
		CheckRedirect: http.DefaultClient.CheckRedirect,
	}

	req, err := http.NewRequest("GET", "https://api.up.com.au/api/v1/transactions", nil)
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
	fmt.Println(string(responseData))

}
