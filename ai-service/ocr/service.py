class OCRService:
    """OCR text extraction from images/scanned documents (Phase 2)."""

    def extract_text(self, file_bytes: bytes, file_type: str) -> str:
        raise NotImplementedError("OCR will be implemented in phase 2")
