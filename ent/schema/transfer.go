package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Transfer struct {
	ent.Schema
}

func (Transfer) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").StorageKey("id").Comment("ID of the transfer used as primary key"),
		field.String("from_address").NotEmpty().Comment("Address of the sender"),
		field.String("to_address").NotEmpty().Comment("Address of the receiver"),
		field.String("token").NotEmpty().Comment("Token associated with the transfer"),
		field.Float("amount").Positive().Comment("Amount transferred"),
		field.String("denom").NotEmpty().Comment("Denomination of the transferred amount"),
		field.Time("created_at").Default(time.Now).Immutable().Comment("Creation time of the transfer"),
	}
}

func (Transfer) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("account", Account.Type).
			Ref("transfers").
			Unique(),
	}
}
