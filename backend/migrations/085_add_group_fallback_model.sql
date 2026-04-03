-- 分组模型别名兜底模型：当请求模型未匹配到任何别名规则时，自动映射到此模型
ALTER TABLE groups ADD COLUMN IF NOT EXISTS fallback_model VARCHAR(100) NOT NULL DEFAULT '';
