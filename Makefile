PROMU_VERSION := 0.6.1

setup_promu:
	curl -s -L https://github.com/prometheus/promu/releases/download/v$(PROMU_VERSION)/promu-$(PROMU_VERSION).linux-amd64.tar.gz | tar -xvzf - 
	mv promu-$(PROMU_VERSION).linux-amd64/promu .

build: 
	./promu build --prefix $(PREFIX) $(PROMU_BINARIES)

test:
	go test