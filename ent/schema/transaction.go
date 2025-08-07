package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Transaction holds the schema definition for the Transaction entity.
type Transaction struct {
	ent.Schema
}

type GasFee struct {
	Amount string `json:"amount"` // Amount of gas fee
	Denom  string `json:"denom"`  // Denomination of the gas fee
}
type Message struct {
	Route   string      `json:"route"`   // Route of the message
	TypeUrl string      `json:"typeUrl"` // Type URL of the message
	Value   interface{} `json:"value"`   // Value of the message, can be of different types
}
type Response struct {
	Log    string   `json:"log"`    // Log of the response
	Info   string   `json:"info"`   // Info of the response
	Error  string   `json:"error"`  // Error message if any
	Data   string   `json:"data"`   // Data of the response
	Events []string `json:"events"` // Events associated with the responsed
}

// Fields of the Transaction.
func (Transaction) Fields() []ent.Field {
	return []ent.Field{
		field.Int("index").Comment("Index of the transaction in the block"),
		field.String("hash").NotEmpty().Comment("Hash of the transaction"),
		field.Bool("success").Default(false).Comment("Whether the transaction was successful"),
		field.Int("block_height").Comment("Height of the block containing the transaction"),
		field.Int("gas_wanted").Comment("Gas wanted for the transaction"),
		field.Int("gas_used").Comment("Gas used by the transaction"),
		field.String("memo").Optional().Comment("Memo of the transaction"),
		field.JSON("gas_fee", []GasFee{}).Optional().Comment("Gas fee paid for the transaction"),
		field.JSON("messages", []Message{}).Optional().Comment("Messages in the transaction"),
		field.JSON("response", Response{}).Optional().Comment("Response of the transaction"),
		field.Time("created_at").Default(time.Now()).Immutable().Comment("Creation time of the transaction"),
	}
}

// Edges of the Transaction.
func (Transaction) Edges() []ent.Edge {
	return nil
}
