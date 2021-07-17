# compile all v2 samples to pipeline job JSON files
.PHONY: all
all:
	@for f in $$(find . -name "*.py"); do \
		compiled="$${f%.*}_pipeline.json"; \
		echo "compiling $${f} to $${compiled}"; \
		dsl-compile-v2 --py "$${f}" --out $${compiled}""; \
	done

# clean up all genereated pipeline job JSON files
.PHONY: clean
clean:
	@find . -name "*.json" -exec rm {} \;
