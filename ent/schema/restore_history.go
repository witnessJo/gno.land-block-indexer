package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type RestoreHistory struct {
	ent.Schema
}

func (RestoreHistory) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Comment("Unique identifier for the restore history entry"),
		field.Int("restore_range_start").Comment("Start of the block restore range"),
		field.Int("restore_range_end").Comment("End of the block restore range"),
		field.Int("being_block").Comment("Block currently being restored"),
	}
}
