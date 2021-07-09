.PHONY: all
all:
	@for f in $$(find . -name "*.py"); do \
		compiled="$${f%.*}_pipeline.json"; \
		echo "compiling $${f} to $${compiled}"; \
		dsl-compile-v2 --py "$${f}" --out $${compiled}""; \
	done

.PHONY: clean
clean:
	@find . -name "*.json" -exec rm {} \;
