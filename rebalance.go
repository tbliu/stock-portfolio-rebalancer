package main

import (
	"encoding/json"
	"net/http"
	"./secrets"
	"io/ioutil"
	"log"
	"fmt"
	"flag"
	"strconv"

	"github.com/alpacahq/alpaca-trade-api-go/alpaca"
)

func getPorfolioAllocation() map[string]float32 {
	// Maps from ticker -> percent
	allocation := map[string]float32 {
		// Hedges
		// "IAU": 0.2,
		// "VTI": 0.2,

		// // Stocks
		// "AMZN": 0.2,
		// "MSFT": 0.2,
		// "ADBE": 0.1,
		"NVDA": 0.50,
		"TSLA": 0.50,
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
		log.Fatal("Error: Response code was %d", resp.StatusCode);
		return nil, fmt.Sprintf("Error: Status code was %d", resp.StatusCode);
	}

	defer resp.Body.Close();

	body, _ := ioutil.ReadAll(resp.Body);
	positions := make([]*alpaca.Position, len(desiredAllocation));
	json.Unmarshal(body, &positions);

	return positions, "";
}

func rebalancePortfolio(
	endpoint string,
	api_key string,
	api_secret string,
	desiredAllocation map[string]float32,
	allocationTooHigh map[string]float32,
	allocationTooLow map[string]float32,
	positions []*alpaca.Position,
	accountEquity float64,
) (bool, string) {
	// Sell off the positions that have gained in percentage
	for _, position := range positions {

	}

	// Sell off the positions that have decreased in percentage

	return true, "";
}

func submitOrder(
	endpoint string,
	api_key string,
	api_secret string,
	desiredAllocation float32,
	orderType string,
	position *alpaca.Position,
) (bool, string) {
	ticker := position.Symbol;
	currentQuantity := strconv.Atoi(position.Qty);
	desiredQuantity := int(accountEquity * desiredAllocation[ticker]);

	numToSell := currentQuantity - desiredQuantity;
	requestBody, err := json.Marshal(map[string]string {
		"symbol": ticker,
			"qty": strconv.Itoa(numToSell),
			"side": "sell",
			"type": "market",
			"time_in_force": "day",
		});

	if err != nil {
		log.Fatal(err);
		return nil, fmt.Sprintf("Error: %s", err.Error());
	}

	req, _ := http.NewRequest("POST", endpoint + "/orders", requestBody);
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
}

func main() {
	dryRunPtr := flag.Bool("dryRun", false, "specifies whether or not this run is a dry run");
	dryRun := *dryRunPtr;

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
		fmt.Println(err);
		return;
	}

	accountEquity, _ := account.Equity.Float64();

	// Maps ticker to actual allocation
	allocationTooHigh := make(map[string]float32);
	allocationTooLow := make(map[string]float32);

	positions, err := genAccountPositions(ENDPOINT, API_KEY, API_SECRET, desiredAllocation);
	if err != "" {
		fmt.Println(err);
		return;
	}

	shouldRebalance := false;

	// Iterate through portfolio to see if desired vs. actual allocation
	// deviates by 5% or more
	for _, position := range positions {
		ticker := position.Symbol;
		desiredAllocationForTicker, ok := desiredAllocation[ticker];
		if !ok {
			err = "Error: " + ticker + " exists in portfolio but not in desired allocation";
			fmt.Println(err);
			return;
		} else {
			valueRangeForTickerLowerBound := float64((desiredAllocationForTicker - 0.05)) *
				accountEquity;
			
			valueRangeForTickerUpperBound := float64((desiredAllocationForTicker + 0.05)) *
				accountEquity;
			
			actualEquityForTicker, _ := position.MarketValue.Float64()
			desiredEquityForTicker, _ := desiredAllocationForTicker * accountEquity;

			if actualEquityForTicker <= valueRangeForTickerLowerBound ||
				actualEquityForTicker >= valueRangeForTickerUpperBound {
				shouldRebalance = true;
			}

			if actualEquityForTicker >= desiredEquityForTicker {
				allocationTooHigh[ticker] = actualEquityForTicker;
			} else {
				allocationTooLow[ticker] = actualEquityForTicker;
			}
		}
	}

	if shouldRebalance {
		fmt.Println("Rebalancing portfolio.");

		success, err := rebalancePortfolio(
			ENDPOINT,
			API_KEY,
			API_SECRET,
			desiredAllocation,
			allocationTooHigh,
			allocationTooLow,
			positions,
			accountEquity,
		);

		if success {
			fmt.Println("Successfully rebalanced portfolio");
		} else {
			fmt.Println("Failed to rebalance portfolio: " + err);
		}
		return;
	} else {
		fmt.Println("Did not rebalance portfolio--no allocations deviated by more than 5%.");
		return;
	}
}
