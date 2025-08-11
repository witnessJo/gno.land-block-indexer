package model

import "time"

//	{
//	  "data": {
//	    "getBlocks": [
//	      {
//	        "hash": "4WQMzKlzyz+6jg9lU5zV0RqIv99UB2iHfL45n+XxVJ8=",
//	        "height": 1,
//	        "time": "2025-07-11T15:07:12.696096956Z",
//	        "total_txs": 0,
//	        "num_txs": 0
//			}
//	}
type Block struct {
	Hash     string    `json:"hash"`      // Hash of the block
	Height   int       `json:"height"`    // Height of the block
	Time     time.Time `json:"time"`      // Timestamp of the block
	TotalTxs int       `json:"total_txs"` // Total number of transactions in the block
	NumTxs   int       `json:"num_txs"`   // Number of transactions in the block
}

//	{
//	  "data": {
//	    "getTransactions": [
//	      {
//	        "index": 0,
//	        "hash": "c15NQRD8/MA+aTl71mmhI6j0AmyYDQKVFk4O3wMvUI0=",
//	        "success": true,
//	        "block_height": 6369,
//	        "gas_wanted": 49434,
//	        "gas_used": 44920,
//	        "memo": "",
//	        "gas_fee": {
//	          "amount": 50,
//	          "denom": "ugnot"
//	        },
//	        "messages": [
//	          {
//	            "route": "bank",
//	            "typeUrl": "send",
//	            "value": {
//	              "from_address": "g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5",
//	              "to_address": "g1cuhgyjzwvz5hjec70xvfh4zqfx079c3r8rnrth",
//	              "amount": "100000000ugnot"
//	            }
//	          }
//	        ],
//	        "response": {
//	          "log": "msg:0,success:true,log:,events:[]",
//	          "info": "",
//	          "error": "",
//	          "data": "",
//	          "events": []
//	        }
//	      }
//	    ]
//	  }
//	}
type Transaction struct {
	Index       int       `json:"index"`        // Index of the transaction in the block
	Hash        string    `json:"hash"`         // Hash of the transaction
	Success     bool      `json:"success"`      // Whether the transaction was successful
	BlockHeight int       `json:"block_height"` // Height of the block containing the transaction
	GasWanted   float64   `json:"gas_wanted"`   // Gas wanted for the transaction
	GasUsed     float64   `json:"gas_used"`     // Gas used by the transaction
	Memo        string    `json:"memo"`         // Memo of the transaction
	GasFee      GasFee    `json:"gas_fee"`      // Gas fee paid for the transaction
	Messages    []Message `json:"messages"`     // Messages in the transaction
	Response    Response  `json:"response"`     // Response of the transaction
}

type GasFee struct {
	Amount float64 `json:"amount"` // Amount of gas fee
	Denom  string  `json:"denom"`  // Denomination of the gas fee
}

type Message struct {
	Route   string         `json:"route"`   // Route of the message
	TypeUrl string         `json:"typeUrl"` // Type URL of the message
	Value   map[string]any `json:"value"`   // Value of the message, can be of different types
}

type Response struct {
	Log    string   `json:"log"`    // Log of the response
	Info   string   `json:"info"`   // Info of the response
	Error  string   `json:"error"`  // Error message if any
	Data   string   `json:"data"`   // Data of the response
	Events []string `json:"events"` // Events associated with the response
}
