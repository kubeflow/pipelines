.PHONY: generate-python python clean-python

generate-python:
	$(MAKE) -C .. generate-python

python: generate-python
	cd .. && uv build --package kfp --out-dir sdk/python/dist

clean-python:
	rm -rf python/build python/dist python/kfp.egg-info
