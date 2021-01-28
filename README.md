# Stock Rebalancer
This Go script automatically rebalances your portfolio whenever one of your holdings deviates from its desired allocation by 5% or more. It uses the [Alpaca api](https://github.com/alpacahq/alpaca-trade-api-go), so you will need to have an account set up with them. It will also send emails to you whenever the script is running, letting you know if the script completed successfully and if there was a rebalance or not.

To set and adjust your portfolio, edit the `getPortfolioAllocation()` function of `rebalance.go`. In your secrets file, you will need:
- an API key and secret key from Alpaca
- a paper API key and paper secret key from Alpaca if you plan on testing with a play money account
- the paper endpoint: https://paper-api.alpaca.markets/v2
- the live endpoint: https://api.alpaca.markets
- the data endpoint: https://data.alpaca.markets
- your email address
- your email password
- your email hostname (smtp)
- your email port

Once this is set up and running without errors on your local machine, you can set up a cron job to have this script run every weekday at a specific point in time (I set it so it runs just after market open).
