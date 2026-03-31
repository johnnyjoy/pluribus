class PluribusError(Exception):
    """Base error for the Pluribus SDK."""


class PluribusAPIError(PluribusError):
    """Raised for non-2xx HTTP responses."""

    def __init__(self, method: str, path: str, status_code: int, response_snippet: str) -> None:
        self.method = method
        self.path = path
        self.status_code = status_code
        self.response_snippet = response_snippet
        super().__init__(f"pluribus api error: {method} {path} returned {status_code}: {response_snippet}")
