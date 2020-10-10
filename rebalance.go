package main

import (
    "./login"
    "./secrets"
    "bytes"
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "net/smtp"
    "strconv"

    "github.com/alpacahq/alpaca-trade-api-go/alpaca"
)

func getPorfolioAllocation() map[string]float32 {
    // Maps from ticker -> percent
    allocation := map[string]float32{
        // 80% stocks
        "AMZN": 0.25,
        "MSFT": 0.11,
        "VTI": 0.10,
        "ADBE": 0.14,
        //"NVDA": 0.2,
        "TSLA": 0.1,

        // 10% bonds
        "TLT": 0.10,

        // 10% gold
        "IAU": 0.10,
    }

    return allocation
}

func genAccountInfo(
    endpoint string,
    apiKey string,
    apiSecret string,
) (*alpaca.Account, string) {
    client := &http.Client{}
    req, _ := http.NewRequest("GET", endpoint+"/account", nil)
    req.Header.Add("APCA-API-KEY-ID", apiKey)
    req.Header.Add("APCA-API-SECRET-KEY", apiSecret)
    resp, err := client.Do(req)

    if err != nil {
        log.Fatal(err)
        return nil, fmt.Sprintf("Error: %s", err.Error())
    }

    if resp.StatusCode < 200 || resp.StatusCode > 299 {
        log.Fatal(fmt.Sprintf("Error: Response code was %d at genAccountInfo"), resp.StatusCode)
        return nil, fmt.Sprintf("Error: Status code was %d", resp.StatusCode)
    }

    defer resp.Body.Close()
    body, _ := ioutil.ReadAll(resp.Body)
    account := alpaca.Account{}
    json.Unmarshal(body, &account)

    return &account, ""
}

func genAccountPositions(
    endpoint string,
    apiKey string,
    apiSecret string,
    desiredAllocation map[string]float32,
) ([]*alpaca.Position, string) {
    client := &http.Client{}
    req, _ := http.NewRequest("GET", endpoint+"/positions", nil)
    req.Header.Add("APCA-API-KEY-ID", apiKey)
    req.Header.Add("APCA-API-SECRET-KEY", apiSecret)
    resp, err := client.Do(req)

    if err != nil {
        log.Fatal(err)
        return nil, fmt.Sprintf("Error: %s", err.Error())
    }

    if resp.StatusCode < 200 || resp.StatusCode > 299 {
        log.Fatal(fmt.Sprintf("Error: Response code was %d at genAccountpositions"), resp.StatusCode)
        return nil, fmt.Sprintf("Error: Status code was %d", resp.StatusCode)
    }

    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
    positions := make([]*alpaca.Position, len(desiredAllocation))
    json.Unmarshal(body, &positions)

    return positions, ""
}

func rebalancePortfolio(
    endpoint string,
    apiKey string,
    apiSecret string,
    desiredAllocation map[string]float32,
    allocationTooHigh map[string]float64,
    allocationTooLow map[string]float64,
    positions []*alpaca.Position,
    accountEquity float64,
) (bool, string) {
    // Sell off the positions that have gained in percentage
    for _, position := range positions {
        ticker := position.Symbol
        if _, ok := allocationTooHigh[ticker]; ok {
            didSucceed, err := submitOrder(
                endpoint,
                apiKey,
                apiSecret,
                desiredAllocation[ticker],
                "sell",
                position,
                ticker,
                accountEquity,
                0, // initial adjustment is 0
                5, // five retries
            )

            if !didSucceed {
                return false, err
            }
        }
    }

    // Buy the positions that have decreased in percentage
    for _, position := range positions {
        ticker := position.Symbol
        if _, ok := allocationTooLow[ticker]; ok {
            didSucceed, err := submitOrder(
                endpoint,
                apiKey,
                apiSecret,
                desiredAllocation[ticker],
                "buy",
                position,
                ticker,
                accountEquity,
                0, // initial adjustment is 0
                5, // five retries
            )

            if !didSucceed {
                return false, err
            }
        }
    }

    return true, ""
}

func submitOrder(
    endpoint string,
    apiKey string,
    apiSecret string,
    desiredAllocationForTicker float32,
    orderType string,
    position *alpaca.Position,
    ticker string,
    accountEquity float64,
    adjustment int,
    numRetries int,
) (bool, string) {
    currentPriceOfStock, _ := position.CurrentPrice.Float64()
    currentQuantity, _ := position.Qty.Float64()
    desiredQuantity := int(accountEquity * float64(desiredAllocationForTicker) / currentPriceOfStock)

    var numToBuyOrSell int
    if orderType == "buy" {
        numToBuyOrSell = desiredQuantity - int(currentQuantity)
    } else if orderType == "sell" {
        numToBuyOrSell = int(currentQuantity) - desiredQuantity
    } else {
        log.Fatal(fmt.Sprintf("Error: invalid transaction given in submitOrder: %s", orderType))
        return false, "Error: invalid transaction given in submitOrder: %s"
    }

    requestBody, err := json.Marshal(map[string]string{
        "symbol":        ticker,
        "qty":           strconv.Itoa(numToBuyOrSell + adjustment),
        "side":          orderType,
        "type":          "market",
        "time_in_force": "day",
    })

    if err != nil {
        log.Fatal(err)
        return false, fmt.Sprintf("Error: %s", err.Error())
    }

    client := &http.Client{}
    req, _ := http.NewRequest("POST", endpoint+"/orders", bytes.NewReader(requestBody))
    req.Header.Add("APCA-API-KEY-ID", apiKey)
    req.Header.Add("APCA-API-SECRET-KEY", apiSecret)
    resp, err := client.Do(req)
    defer resp.Body.Close()

    if err != nil {
        log.Fatal(err)
        return false, fmt.Sprintf("Error: %s", err.Error())
    }

    if resp.StatusCode == 403 {
        // Buying power is not sufficient, so try again with fewer shares
        if numRetries == 0 {
            log.Fatal(fmt.Sprintf("Error: Response code was %d at submitOrder even after retries", resp.StatusCode))
            return false, fmt.Sprintf("Error: Response code was %d at submitOrder even after retries")
        }

        return submitOrder(
            endpoint,
            apiKey,
            apiSecret,
            desiredAllocationForTicker,
            orderType,
            position,
            ticker,
            accountEquity,
            adjustment - 1,
            numRetries - 1,
        )
    } else if resp.StatusCode < 200 || resp.StatusCode > 299 {
        log.Fatal(fmt.Sprintf("Error: Response code was %d at submitOrder", resp.StatusCode))
        return false, fmt.Sprintf("Error: Status code was %d", resp.StatusCode)
    }

    return true, ""
}

func sendEmailOnCompletion(msg string) {
    auth := login.LoginAuth(secrets.EMAIL_ADDR, secrets.EMAIL_PASSWORD)
    from := secrets.EMAIL_ADDR
    recipients := []string{secrets.EMAIL_ADDR}
    body := []byte(
        "From: " + from + "\n" +
            "To: " + recipients[0] + "\n" +
            "Subject: Stock rebalancer\n\n" +
            msg,
    )

    err := smtp.SendMail(secrets.EMAIL_HOSTNAME+secrets.EMAIL_PORT, auth, from, recipients, body)

    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Email sent.")
    return
}

func main() {
    dryRunPtr := flag.Bool("test", false, "specifies whether or not this run is a dry run")
    flag.Parse()
    dryRun := *dryRunPtr
    fmt.Println(dryRun)

    desiredAllocation := getPorfolioAllocation()

    var APIKey string
    var APISecret string
    var Endpoint string

    if dryRun {
        APIKey = secrets.PAPER_API_KEY_ID
        APISecret = secrets.PAPER_SECRET_KEY
        Endpoint = secrets.PAPER_ENDPOINT
    } else {
        APIKey = secrets.API_KEY_ID
        APISecret = secrets.SECRET_KEY
        Endpoint = secrets.LIVE_ENDPOINT
    }

    // Get account equity
    account, err := genAccountInfo(Endpoint, APIKey, APISecret)
    if err != "" {
        fmt.Println(err)
        sendEmailOnCompletion(err)
        return
    }

    accountEquity, _ := account.Equity.Float64()

    // Maps ticker to actual allocation
    allocationTooHigh := make(map[string]float64)
    allocationTooLow := make(map[string]float64)

    positions, err := genAccountPositions(Endpoint, APIKey, APISecret, desiredAllocation)
    if err != "" {
        fmt.Println(err)
        sendEmailOnCompletion(err)
        return
    }

    shouldRebalance := false

    // Iterate through portfolio to see if desired vs. actual allocation
    // deviates by 5% or more
    for _, position := range positions {
        ticker := position.Symbol
        desiredAllocationForTicker, ok := desiredAllocation[ticker]
        if !ok {
            err = "Error: " + ticker + " exists in portfolio but not in desired allocation"
            fmt.Println(err)
            return
        }

        valueRangeForTickerLowerBound := float64((desiredAllocationForTicker - 0.05)) *
            accountEquity

        valueRangeForTickerUpperBound := float64((desiredAllocationForTicker + 0.05)) *
            accountEquity

        actualEquityForTicker, _ := position.MarketValue.Float64()
        desiredEquityForTicker := float64(desiredAllocationForTicker) * accountEquity

        if actualEquityForTicker <= valueRangeForTickerLowerBound ||
            actualEquityForTicker >= valueRangeForTickerUpperBound {
            shouldRebalance = true
        }

        if actualEquityForTicker >= desiredEquityForTicker {
            allocationTooHigh[ticker] = actualEquityForTicker
        } else {
            allocationTooLow[ticker] = actualEquityForTicker
        }     
    }

    if shouldRebalance {
        fmt.Println("Rebalancing portfolio.")

        success, err := rebalancePortfolio(
            Endpoint,
            APIKey,
            APISecret,
            desiredAllocation,
            allocationTooHigh,
            allocationTooLow,
            positions,
            accountEquity,
        )

        if success {
            fmt.Println("Successfully rebalanced portfolio")
            sendEmailOnCompletion("Successfully rebalanced portfolio")
        } else {
            fmt.Println("Failed to rebalance portfolio: " + err)
            sendEmailOnCompletion("Failed to rebalance portfolio: " + err)
        }
        return
    }
    fmt.Println("Did not rebalance portfolio--no allocations deviated by more than 5%.")
    sendEmailOnCompletion("Did not rebalance portfolio--no allocations deviated by more than 5%.")
    return
}
