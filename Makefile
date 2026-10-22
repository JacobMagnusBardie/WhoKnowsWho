.PHONY: spec spec-check

spec:
	cd backend && go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go -d . -o ./docs

spec-check: spec
	git diff --exit-code -- backend/docs/