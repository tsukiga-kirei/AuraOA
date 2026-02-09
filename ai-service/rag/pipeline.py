class RAGPipeline:
    """RAG pipeline for document vectorization and retrieval (Phase 2)."""

    def ingest_document(self, tenant_id: str, document: dict) -> None:
        raise NotImplementedError("RAG ingest will be implemented in phase 2")

    def retrieve(self, tenant_id: str, query: str, top_k: int = 5) -> list[dict]:
        raise NotImplementedError("RAG retrieve will be implemented in phase 2")
