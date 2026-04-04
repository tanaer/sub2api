package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

// SubscriptionPlan holds the schema definition for the SubscriptionPlan entity.
type SubscriptionPlan struct {
	ent.Schema
}

func (SubscriptionPlan) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "subscription_plans"},
	}
}

func (SubscriptionPlan) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (SubscriptionPlan) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(100).
			Comment("plan name"),
		field.String("description").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Comment("描述"),
		field.Int64("group_id").
			Optional().
			Nillable().
			Comment("可选绑定分组"),
		field.String("billing_mode").
			MaxLen(20).
			Default("per_request").
			Comment("计费模式: per_request / per_usd"),
		field.Int64("request_quota").
			Default(0).
			Comment("按次配额（billing_mode=per_request 时使用）"),
		field.Float("daily_limit_usd").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}).
			Default(0).
			Comment("每日 USD 限额（billing_mode=per_usd 时使用）"),
		field.Float("weekly_limit_usd").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}).
			Default(0).
			Comment("每周 USD 限额"),
		field.Float("monthly_limit_usd").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}).
			Default(0).
			Comment("每月 USD 限额"),
		field.Int("validity_days").
			Default(30).
			Comment("有效期（天）"),
		field.String("status").
			MaxLen(20).
			Default("active").
			Comment("状态: active / archived"),
	}
}

func (SubscriptionPlan) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("group", Group.Type).
			Ref("subscription_plans").
			Field("group_id").
			Unique(),
		edge.To("redeem_codes", RedeemCode.Type),
	}
}

func (SubscriptionPlan) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("group_id"),
		index.Fields("billing_mode"),
	}
}
