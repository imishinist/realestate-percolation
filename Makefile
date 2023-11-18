
.PHONY: setup
setup: build
	@make setup-realestate
	@make setup-realestate_query
	@make setup-station

setup-%:config/%.mapping.json
	@echo "setup \"$*\""
	@make put-index-$*
	@make seed-$*
	@echo

put-index-%:config/%.mapping.json
	@echo "put index $*"
	@./bin/put-index.sh $* $<

seed-%:seed/%.jsonl build
	@echo "seed $*"
	@./bin/esutil -index $* < seed/$*.jsonl

.PHONY: build
build:
	@go build -o bin/esutil ./tools/esutil
