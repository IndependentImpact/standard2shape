# Separate authoring from hosted execution

`standard2shape` authors and locally validates canonical standard packages but does not provide a separate hosted validation service. Local tooling and `ii-backend` must invoke one reusable validation interface over the same immutable packages and conformance suites, while `ii-backend` owns hosted access, storage, governance, orchestration, and attestations; this avoids duplicating the II Platform while preventing deployment-specific semantic drift.
