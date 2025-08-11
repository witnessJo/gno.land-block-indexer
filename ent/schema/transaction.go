// Transaction.go - Transaction 스키마
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Transaction holds the schema definition for the Transaction entity.
type Transaction struct {
	ent.Schema
}

type GasFee struct {
	Amount float64 `json:"amount"`
	Denom  string  `json:"denom"`
}

type Message struct {
	Route   string         `json:"route"`
	TypeUrl string         `json:"typeUrl"`
	Value   map[string]any `json:"value"`
}

type Response struct {
	Log    string   `json:"log"`
	Info   string   `json:"info"`
	Error  string   `json:"error"`
	Data   string   `json:"data"`
	Events []string `json:"events"`
}

// Fields of the Transaction.
func (Transaction) Fields() []ent.Field {
	return []ent.Field{
		field.Int("index").Comment("Index of the transaction in the block"),
		field.String("hash").NotEmpty().Comment("Hash of the transaction"),
		field.Bool("success").Default(false).Comment("Whether the transaction was successful"),
		field.Int("block_height").Comment("Height of the block containing the transaction"),
		field.Float("gas_wanted").Comment("Gas wanted for the transaction"),
		field.Float("gas_used").Comment("Gas used by the transaction"),
		field.String("memo").Optional().Comment("Memo of the transaction"),
		field.JSON("gas_fee", GasFee{}).Optional().Comment("Gas fee paid for the transaction"),
		field.JSON("messages", []Message{}).Optional().Comment("Messages in the transaction"),
		field.JSON("response", Response{}).Optional().Comment("Response of the transaction"),
		field.Time("created_at").Default(time.Now()).Immutable().Comment("Creation time of the transaction"),
	}
}

// Edges of the Transaction.
func (Transaction) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("block", Block.Type).
			Ref("transactions").
			Unique(),
	}
}
