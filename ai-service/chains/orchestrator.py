from models.schemas import (
    AuditResponse,
    ChecklistResult,
    KBMode,
    MergedRule,
    AIConfig,
)


class ChainOrchestrator:
    """Routes audit requests to the appropriate chain based on KB mode."""

    def execute_audit(
        self,
        form_data: dict,
        rules: list[MergedRule],
        kb_mode: KBMode,
        ai_config: AIConfig,
    ) -> AuditResponse:
        if kb_mode == KBMode.RULES_ONLY:
            return self._run_checklist_chain(form_data, rules, ai_config)
        elif kb_mode == KBMode.RAG_ONLY:
            return self._run_retrieval_chain(form_data)
        elif kb_mode == KBMode.HYBRID:
            return self._run_hybrid(form_data, rules)
        raise ValueError(f"Unknown kb_mode: {kb_mode}")

    def _run_checklist_chain(
        self, form_data: dict, rules: list[MergedRule], ai_config: AIConfig
    ) -> AuditResponse:
        """Execute structured checklist rules one by one (Phase 1 core)."""
        results: list[ChecklistResult] = []
        reasoning_parts: list[str] = []

        for r in rules:
            # Placeholder: real LLM call will replace this
            passed = True
            reason = f"Rule '{r.content}' checked against form data."
            results.append(
                ChecklistResult(rule_id=r.id, passed=passed, reasoning=reason)
            )
            reasoning_parts.append(f"[{r.id}] {reason}")

        all_passed = all(r.passed for r in results)
        recommendation = "approve" if all_passed else "revise"

        return AuditResponse(
            recommendation=recommendation,
            details=results,
            ai_reasoning="\n".join(reasoning_parts),
        )

    def _run_retrieval_chain(self, form_data: dict) -> AuditResponse:
        """RAG retrieval chain (Phase 2)."""
        raise NotImplementedError("RAG mode will be implemented in phase 2")

    def _run_hybrid(self, form_data: dict, rules: list[MergedRule]) -> AuditResponse:
        """Hybrid chain combining checklist + RAG (Phase 2)."""
        raise NotImplementedError("Hybrid mode will be implemented in phase 2")
