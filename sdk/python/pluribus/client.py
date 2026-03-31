from __future__ import annotations

from typing import Any, Dict, List, Optional

import requests

from .exceptions import PluribusAPIError


class PluribusClient:
    """HTTP client focused on the agent loop: recall_context → work → record_experience."""

    def __init__(self, base_url: str, api_key: Optional[str] = None, timeout: float = 15.0) -> None:
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.timeout = timeout

    def recall_context(
        self,
        query: str,
        *,
        tags: Optional[List[str]] = None,
        correlation_id: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Load governing context before you do meaningful work (maps to MCP tool recall_context)."""
        body: Dict[str, Any] = {"retrieval_query": query}
        if tags:
            body["tags"] = tags
        if correlation_id:
            body["correlation_id"] = correlation_id
        return self._request_json("POST", "/v1/recall/compile", body)

    def record_experience(
        self,
        summary: str,
        *,
        tags: Optional[List[str]] = None,
        entities: Optional[List[str]] = None,
        correlation_id: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Log what happened after meaningful work; advisory until promoted (maps to record_experience)."""
        body: Dict[str, Any] = {"summary": summary, "source": "mcp"}
        if tags:
            body["tags"] = tags
        if entities:
            body["entities"] = entities
        if correlation_id:
            body["correlation_id"] = correlation_id
        return self._request_json("POST", "/v1/advisory-episodes", body)

    def list_pending_candidates(self) -> List[Dict[str, Any]]:
        return self._request_json("GET", "/v1/curation/pending")

    def review_candidate(self, candidate_id: str) -> Dict[str, Any]:
        if not candidate_id.strip():
            raise ValueError("candidate_id is required")
        return self._request_json("GET", f"/v1/curation/candidates/{candidate_id}/review")

    def promote_candidate(self, candidate_id: str) -> Dict[str, Any]:
        if not candidate_id.strip():
            raise ValueError("candidate_id is required")
        return self._request_json("POST", f"/v1/curation/candidates/{candidate_id}/materialize", {})

    def _request_json(self, method: str, path: str, json_body: Optional[Dict[str, Any]] = None) -> Any:
        url = f"{self.base_url}{path}"
        headers = {"Content-Type": "application/json"}
        if self.api_key:
            headers["X-API-Key"] = self.api_key

        response = requests.request(
            method=method,
            url=url,
            json=json_body,
            headers=headers,
            timeout=self.timeout,
        )

        if not (200 <= response.status_code < 300):
            text = (response.text or "").strip()
            snippet = text[:400] + ("..." if len(text) > 400 else "")
            raise PluribusAPIError(method, path, response.status_code, snippet)

        if not response.content:
            return None
        return response.json()
