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
		edge.To("transactions", Transaction.Type),
		edge.To("transfers", Transfer.Type),
	}
}
