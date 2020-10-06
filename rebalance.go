package main

import (
	"encoding/json"
	"net/http"
	"./secrets"
	"io/ioutil"
	"log"
	"fmt"

	"github.com/alpacahq/alpaca-trade-api-go/alpaca"
	// "github.com/alpacahq/alpaca-trade-api-go/common"
	// "github.com/shopspring/decimal"
)

func getPorfolioAllocation() map[string]float32 {
	// Maps from ticker -> percent
	allocation := map[string]float32 {
		// Hedges
		"IAU": 0.2,
		"VTI": 0.2,

		// Stocks
		"AMZN": 0.2,
		"MSFT": 0.2,
		"ADBE": 0.1,
		"NVDA": 0.1,
	};

	return allocation;
}

func genAccountInfo(
	endpoint string,
	api_key string,
	api_secret string,
) (*alpaca.Account, string) {
	client := &http.Client {};
	req, _ := http.NewRequest("GET", endpoint + "/account", nil);
	req.Header.Add("APCA-API-KEY-ID", api_key);
	req.Header.Add("APCA-API-SECRET-KEY", api_secret);
	resp, err := client.Do(req);

	if err != nil {
		log.Fatal(err);
		return nil, fmt.Sprintf("Error: %s", err.Error());
	}
	
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Fatal("Error: Response code was %d", resp.StatusCode);
		return nil, fmt.Sprintf("Error: Status code was %d", resp.StatusCode);
	}
	
	defer resp.Body.Close();
	body, _ := ioutil.ReadAll(resp.Body);
	account := alpaca.Account {};
	json.Unmarshal(body, &account);

	return &account, "";
}

func genAccountPositions(
	endpoint string,
	api_key string,
	api_secret string,
	desiredAllocation map[string]float32,
) ([]*alpaca.Position, string) {
	client := &http.Client {};
	req, _ := http.NewRequest("GET", endpoint + "/positions", nil);
	req.Header.Add("APCA-API-KEY-ID", api_key);
	req.Header.Add("APCA-API-SECRET-KEY", api_secret);
	resp, err := client.Do(req);

	if err != nil {
		log.Fatal(err);
		return nil, fmt.Sprintf("Error: %s", err.Error());
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Fatal("Error: Respones code was %d", resp.StatusCode);
		return nil, fmt.Sprintf("Error: Status code was %d", resp.StatusCode);
	}

	defer resp.Body.Close();

	body, _ := ioutil.ReadAll(resp.Body);
	positions := make([]*alpaca.Position, len(desiredAllocation));
	json.Unmarshal(body, &positions);

	return positions, "";
}

func rebalance(dryRun bool) (int, string) {
	desiredAllocation := getPorfolioAllocation();

	var API_KEY string;
	var API_SECRET string;
	var ENDPOINT string;

	if dryRun {
		API_KEY = secrets.PAPER_API_KEY_ID;
		API_SECRET = secrets.PAPER_SECRET_KEY;
		ENDPOINT = secrets.PAPER_ENDPOINT;
	} else {
		API_KEY = secrets.API_KEY_ID;
		API_SECRET = secrets.SECRET_KEY;
		ENDPOINT = secrets.LIVE_ENDPOINT;
	}

	// Get account equity
	account, err := genAccountInfo(ENDPOINT, API_KEY, API_SECRET);
	if err != "" {
		return 1, err;
	}

	accountEquity := account.Equity;

	fmt.Println("Account equity", accountEquity);

	// Iterate through portfolio to see if desired vs. actual allocation
	// deviates by 5% or more
	// actualAllocation := make(map[string]float32);

	positions, err := genAccountPositions(ENDPOINT, API_KEY, API_SECRET, desiredAllocation);
	if err != "" {
		return 1, err;
	}
	fmt.Println(positions);
	return 0, "";

	// for ticker, allocationPercentage := range desiredAllocation {

	// }
}

func main() {
	rebalance(true);
}
