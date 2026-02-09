from pydantic import BaseModel
from enum import Enum


class KBMode(str, Enum):
    RULES_ONLY = "rules_only"
    RAG_ONLY = "rag_only"
    HYBRID = "hybrid"


class RuleScope(str, Enum):
    MANDATORY = "mandatory"
    DEFAULT_ON = "default_on"
    DEFAULT_OFF = "default_off"


class MergedRule(BaseModel):
    id: str
    content: str
    scope: RuleScope
    source: str  # tenant | user
    is_locked: bool = False
    priority: int = 0


class AIConfig(BaseModel):
    model_provider: str = "local"
    model_name: str = "default"
    prompt_template: str | None = None
    context_window_size: int = 4096


class FormField(BaseModel):
    name: str
    value: str


class AuditRequest(BaseModel):
    form_data: dict
    rules: list[MergedRule]
    kb_mode: KBMode
    ai_config: AIConfig
    trace_id: str | None = None


class ChecklistResult(BaseModel):
    rule_id: str
    passed: bool
    reasoning: str


class AuditResponse(BaseModel):
    recommendation: str  # approve | reject | revise
    details: list[ChecklistResult]
    ai_reasoning: str
    trace_id: str | None = None
