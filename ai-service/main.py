import logging
from fastapi import FastAPI, Request
from models.schemas import AuditRequest, AuditResponse, KBMode
from chains.orchestrator import ChainOrchestrator

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("ai-service")

app = FastAPI(title="OA Smart Audit - AI Service", version="0.1.0")
orchestrator = ChainOrchestrator()


@app.get("/health")
async def health():
    return {"status": "ok", "service": "ai-audit-service"}


@app.post("/api/audit", response_model=AuditResponse)
async def execute_audit(request: AuditRequest, req: Request):
    # Extract trace ID from header or request body
    trace_id = req.headers.get("X-Trace-ID", request.trace_id)
    if trace_id:
        logger.info(f"[Trace: {trace_id}] Audit request received, kb_mode={request.kb_mode}")

    result = orchestrator.execute_audit(
        form_data=request.form_data,
        rules=request.rules,
        kb_mode=request.kb_mode,
        ai_config=request.ai_config,
    )
    result.trace_id = trace_id

    if trace_id:
        logger.info(f"[Trace: {trace_id}] Audit complete, recommendation={result.recommendation}")

    return result
