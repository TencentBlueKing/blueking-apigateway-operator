build: ./bin
	@if [ ! -e "./bin" ]; then mkdir bin; fi
	go build -o ./bin/test-operator .

container: build
	docker build . -t test-operator:latest

run: container crdDepend
	kubectl apply -f deploy.yaml

update: container crdDepend
	kubectl scale deploy test-operator --replicas=0
	kubectl scale deploy test-operator --replicas=1

crdDepend: ../../../config/crd/bases
	cd ../../../config/crd/bases && for crd in `ls`; do kubectl apply -f $$crd; done

stop:
	kubectl delete -f deploy.yaml