fuseki-data:
	docker volume create fuseki-data

fuseki:
	docker run --restart always --name fuseki -p 3030:3030 --volume fuseki-data:/fuseki stain/jena-fuseki

run-deployed:
	docker run --rm -p3030:3030 -it ghcr.io/Senforsce/sparql

build:
	docker build . -t Senforsce/sparql

run-built:
	docker run --rm -p3030:3030 -it Senforsce/sparql
