.PHONY: tracer verify

tracer:
	npm install
	npm run tracer

verify:
	npm ci
	npm run verify
