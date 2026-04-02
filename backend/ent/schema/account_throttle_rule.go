package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AccountThrottleRule 账户智能限流规则
//
// 根据上游错误响应自动限流账号：
//   - 匹配条件：关键词列表 + 匹配模式（包含/精确）
//   - 触发方式：立即限流（1次匹配）/ 累计限流（窗口内N次）
//   - 限流方式：时长限流（N秒）/ 定期恢复（每日指定时刻）
//   - 平台范围：规则适用的平台
type AccountThrottleRule struct {
	ent.Schema
}

func (AccountThrottleRule) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "account_throttle_rules"},
	}
}

func (AccountThrottleRule) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (AccountThrottleRule) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(100).
			NotEmpty(),

		field.Bool("enabled").
			Default(true),

		field.Int("priority").
			Default(0),

		// keywords: 关键词列表（OR关系）
		field.JSON("keywords", []string{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),

		// match_mode: "contains"（包含）/ "exact"（精确）
		field.String("match_mode").
			MaxLen(20).
			Default("contains"),

		// trigger_mode: "immediate"（立即）/ "accumulated"（累计）
		field.String("trigger_mode").
			MaxLen(20).
			Default("immediate"),

		// accumulated_count: 累计触发阈值（trigger_mode=accumulated 时使用）
		field.Int("accumulated_count").
			Default(3),

		// accumulated_window: 累计窗口（秒）（trigger_mode=accumulated 时使用）
		field.Int("accumulated_window").
			Default(60),

		// action_type: "duration"（时长限流）/ "scheduled_recovery"（定期恢复）
		field.String("action_type").
			MaxLen(30).
			Default("duration"),

		// action_duration: 限流时长（秒）（action_type=duration 时使用）
		field.Int("action_duration").
			Default(300),

		// action_recover_hour: 恢复时刻（0-23）（action_type=scheduled_recovery 时使用）
		field.Int("action_recover_hour").
			Default(0),

		// platforms: 适用平台列表，空表示所有平台
		field.JSON("platforms", []string{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),

		field.Text("description").
			Optional().
			Nillable(),
	}
}

func (AccountThrottleRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("enabled"),
		index.Fields("priority"),
	}
}
