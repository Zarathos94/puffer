package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/Zarathos94/puffer/etherscanclient"
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

// GetTransactionsByAddress wraps etherscanclient.GetTransactionsByAddress for use in utils
func GetTransactionsByAddress(address string, startBlock, endBlock, page, offset int, sort string) ([]etherscanclient.Transaction, error) {
	return etherscanclient.GetTransactionsByAddress(address, startBlock, endBlock, page, offset, sort)
}
