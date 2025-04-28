package etherscanclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func getEtherscanAPIKey() string {
	return os.Getenv("ETHERSCAN_API_KEY")
}

func GetBlockNumberByTimestamp(ts int64) (string, error) {
	apiKey := getEtherscanAPIKey()
	url := fmt.Sprintf(
		"https://api.etherscan.io/api?module=block&action=getblocknobytime&timestamp=%d&closest=before&apikey=%s",
		ts, apiKey,
	)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Status != "1" {
		return "", fmt.Errorf("etherscan error: %s", result.Message)
	}
	return result.Result, nil
}

func CallContractAtBlock(contract, data, block string) (string, error) {
	apiKey := getEtherscanAPIKey()
	url := fmt.Sprintf(
		"https://api.etherscan.io/api?module=proxy&action=eth_call&to=%s&data=%s&tag=%s&apikey=%s",
		contract, data, block, apiKey,
	)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		Jsonrpc string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Result, nil
}

// Transaction represents a single transaction returned by Etherscan
// You can expand this struct as needed
// See: https://docs.etherscan.io/api-endpoints/accounts#get-a-list-of-normal-transactions-by-address
type Transaction struct {
	BlockNumber       string `json:"blockNumber"`
	TimeStamp         string `json:"timeStamp"`
	Hash              string `json:"hash"`
	Nonce             string `json:"nonce"`
	BlockHash         string `json:"blockHash"`
	TransactionIndex  string `json:"transactionIndex"`
	From              string `json:"from"`
	To                string `json:"to"`
	Value             string `json:"value"`
	Gas               string `json:"gas"`
	GasPrice          string `json:"gasPrice"`
	IsError           string `json:"isError"`
	TxReceiptStatus   string `json:"txreceipt_status"`
	Input             string `json:"input"`
	ContractAddress   string `json:"contractAddress"`
	CumulativeGasUsed string `json:"cumulativeGasUsed"`
	GasUsed           string `json:"gasUsed"`
	Confirmations     string `json:"confirmations"`
}

// GetTransactionsByAddress fetches a list of transactions for a given address from Etherscan
func GetTransactionsByAddress(address string, startBlock, endBlock, page, offset int, sort string) ([]Transaction, error) {
	apiKey := getEtherscanAPIKey()
	url := fmt.Sprintf(
		"https://api.etherscan.io/api?module=account&action=txlist&address=%s&startblock=%d&endblock=%d&page=%d&offset=%d&sort=%s&apikey=%s",
		address, startBlock, endBlock, page, offset, sort, apiKey,
	)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		Status  string          `json:"status"`
		Message string          `json:"message"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	if raw.Status != "1" {
		// Try to extract the result as a string for more info
		var resultMsg string
		_ = json.Unmarshal(raw.Result, &resultMsg)
		return nil, fmt.Errorf("etherscan error: %s (%s)", raw.Message, resultMsg)
	}

	var txs []Transaction
	if err := json.Unmarshal(raw.Result, &txs); err == nil {
		return txs, nil
	}
	// If not an array, check if it's a string (e.g., "No transactions found")
	var s string
	if err := json.Unmarshal(raw.Result, &s); err == nil {
		return []Transaction{}, nil
	}
	return nil, fmt.Errorf("unexpected result format from etherscan")
}
