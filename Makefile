.PHONY: test e2e

test:
	$(MAKE) -C apps/server test

e2e:
	$(MAKE) -C apps/e2e test
