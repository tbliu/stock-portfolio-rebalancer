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
	client := &http.Client {};
	req, _ := http.NewRequest("GET", ENDPOINT + "/account", nil);
	req.Header.Add("APCA-API-KEY-ID", API_KEY);
	req.Header.Add("APCA-API-SECRET-KEY", API_SECRET);
	resp, err := client.Do(req);

	if err != nil {
		log.Fatal(err);
		return 1, fmt.Sprintf("Error: %s", err.Error());
	}
	
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Fatal("Error: Response code was %d", resp.StatusCode);
		return 1, fmt.Sprintf("Error: Status code was %d", resp.StatusCode);
	}
	
	defer resp.Body.Close();
	body, _ := ioutil.ReadAll(resp.Body);
	account := alpaca.Account {};
	json.Unmarshal(body, &account);
	accountEquity := account.Equity;

	// Iterate through portfolio to see if desired vs. actual allocation
	// deviates by 5% or more
	actualAllocation := make(map[string]float32);
	req, _ = http.NewRequest("GET", ENDPOINT + "/positions", nil);
	req.Header.Add("APCA-API-KEY-ID", API_KEY);
	req.Header.Add("APCA-API-SECRET-KEY", API_SECRET);
	resp, err = client.Do(req);

	if err != nil {
		log.Fatal(err);
	}

	var positions [len(desiredAllocation)] alpaca.Position;

	for ticker, allocationPercentage := range desiredAllocation {

	}
}

func main() {
}
