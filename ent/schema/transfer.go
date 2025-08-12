package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Transfer struct {
	ent.Schema
}

func (Transfer) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").StorageKey("id").Comment("ID of the transfer used as primary key"),
		field.String("hash").NotEmpty().Comment("Hash of the transfer transaction"),
		field.String("func").NotEmpty().Comment("Function name of the transfer"),
		field.String("from_address").Optional().StorageKey("from_address").Comment("Address of the sender"),
		field.String("to_address").Optional().Comment("Address of the receiver"),
		field.String("token").NotEmpty().Comment("Token associated with the transfer"),
		field.Float("amount").Positive().Comment("Amount transferred"),
		field.String("denom").NotEmpty().Comment("Denomination of the transferred amount"),
		field.Time("created_at").Default(time.Now).Immutable().Comment("Creation time of the transfer"),
	}
}

func (Transfer) Edges() []ent.Edge {
	return []ent.Edge{}
}

func (Transfer) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("from_address", "token"),
		index.Fields("to_address", "token"),
	}
}
