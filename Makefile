.PHONY: build test validate lint split clean fmt snapshots-push

FERROCAT = go run ./cmd/ferrocat

build:
	$(FERROCAT) build --output dist/

test:
	go test -race -count=1 ./...

validate:
	$(FERROCAT) validate

lint:
	golangci-lint run ./...

split:
	@test -n "$(INPUT)" || (echo "Usage: make split INPUT=/path/to/catalog.json" && exit 1)
	$(FERROCAT) split $(INPUT) --output providers/

clean:
	rm -rf dist/

# Archive the content-addressed price snapshots produced by
# `ferrocat snapshot-provenance` onto the orphan `snapshots` branch, keeping the
# (large, churny) raw archive out of main's history. Idempotent: content-addressed
# files that already exist on the branch are re-added as no-ops.
# ponytail: shell + worktree over a Go git wrapper; needs a live remote to verify.
snapshots-push:
	@test -d snapshots || { echo "no snapshots/ dir — run 'ferrocat snapshot-provenance' first"; exit 1; }
	@git fetch origin snapshots 2>/dev/null || true
	git worktree add -f .snapshots-wt snapshots 2>/dev/null \
		|| git worktree add -f --detach .snapshots-wt origin/snapshots 2>/dev/null \
		|| git worktree add -f --orphan .snapshots-wt snapshots
	cp snapshots/*.json .snapshots-wt/
	cd .snapshots-wt && git add -A && \
		(git diff --cached --quiet || git commit -m "snapshots: archive verified price provenance") && \
		git push origin HEAD:snapshots
	git worktree remove -f .snapshots-wt

fmt:
	gofmt -w .
