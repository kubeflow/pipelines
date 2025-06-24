# Generate SVGs from mermaid files (MMD)
# Requires: https://github.com/mermaid-js/mermaid-cli
#

.PHONY: all clean

all: $(patsubst %.mmd,%.mmd.svg,$(wildcard *.mmd))

%.mmd.svg: %.mmd
	mmdc -i "$<" -o "$@"

clean:
	rm *.mmd.svg
