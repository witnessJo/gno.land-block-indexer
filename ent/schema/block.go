package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Block holds the schema definition for the Block entity.
type Block struct {
	ent.Schema
}

// Fields of the Block.
func (Block) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").StorageKey("height").Comment("Height of the block used as primary key"),
		field.String("hash").NotEmpty().Comment("Hash of the block"),
		field.Time("time").Comment("Timestamp of the block"),
		field.Int("total_txs").Comment("Total number of transactions in the block"),
		field.Int("num_txs").Comment("Number of transactions in the block"),
		field.Time("created_at").Default(time.Now).Immutable().Comment("Creation time of the block"),
	}
}

// Edges of the Block.
func (Block) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("transactions", Transaction.Type),
	}
}
