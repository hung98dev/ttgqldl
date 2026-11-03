# backend/test/golden

Golden gameplay vectors belong here.

**Status: intentionally empty in M01 Phase A.** The authoritative reference
vectors already exist in `tooling/refeval/testdata/golden_vectors.json` and are
checked by `tooling/reference-eval.ps1|.sh`. M02 must reproduce them from the
real reducer and only then add Go golden fixtures here. The independent
evaluator must never be modified to match a backend defect.
