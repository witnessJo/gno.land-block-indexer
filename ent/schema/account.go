package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Block holds the schema definition for the Block entity.
type Account struct {
	ent.Schema
}

// Fields of the Block.
func (Account) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").StorageKey("address").NotEmpty().Comment("Address of the account"),
		field.String("token").NotEmpty().Comment("Token associated with the account"),
		field.Float("amount").Comment("Amount of the token in the account"),
	}
}

// Edges of the Block.
func (Account) Edges() []ent.Edge {
	return []ent.Edge{
		// Allow transfer_to and transfer_from edge to optionals
		edge.To("transfers_to", Transfer.Type).
			StorageKey(edge.Column("to_address")),
		edge.To("transfers_from", Transfer.Type).
			StorageKey(edge.Column("from_address")),
	}
}
