from pluribus import PluribusClient


def main() -> None:
    client = PluribusClient("http://127.0.0.1:8123")

    # 1) recall_context (pre-action)
    bundle = client.recall_context(
        "Refactor auth middleware safely",
        tags=["auth"],
        correlation_id="session-123",
    )

    # 2) plan / reason (use bundle in a real agent)
    _ = bundle

    # 3) act (your tools, edits, commits)
    print(
        "recall:",
        len(bundle.get("governing_constraints", [])),
        "constraints,",
        len(bundle.get("known_failures", [])),
        "failures — act phase.",
    )

    # 4) record_experience (post-action)
    episode = client.record_experience(
        "Fixed race in session refresh; added test coverage.",
        tags=["auth", "incident"],
        correlation_id="session-123",
    )
    print("recorded advisory episode", episode.get("id"), "deduplicated=", episode.get("deduplicated", False))


if __name__ == "__main__":
    main()
