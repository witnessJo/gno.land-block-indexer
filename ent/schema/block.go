package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"time"
)

// Block holds the schema definition for the Block entity.
type Block struct {
	ent.Schema
}

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

// Fields of the Block.
func (Block) Fields() []ent.Field {
	return []ent.Field{
		field.String("hash").NotEmpty().Comment("Hash of the block"),
		field.Int("height").Comment("Height of the block"),
		field.Time("time").Comment("Timestamp of the block"),
		field.Int("total_txs").Comment("Total number of transactions in the block"),
		field.Int("num_txs").Comment("Number of transactions in the block"),
		field.Time("created_at").Default(time.Now).Immutable().Comment("Creation time of the block"),
	}
}

// Edges of the Block.
func (Block) Edges() []ent.Edge {
	return nil
}
