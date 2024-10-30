include .env
.PHONY: build pull push dev down docker helm
BRANCH := ${shell git rev-parse --symbolic-full-name --abbrev-ref HEAD}
NAME_SPACE=tonywangcn
SRV_NAME=ten-million-domains
REPO=ghcr.io
TAG=$(shell date +%Y%m%d%H%M%S)
NAME=${REPO}/${NAME_SPACE}/${SRV_NAME}
export KUBECONFIG=./terraform/kubeconfig-ten-million-domains-iad.yaml


build:
	echo build ${SRV_NAME}:latest
	cp ./docker/Dockerfile .
	docker build -t ${SRV_NAME}:latest .
	rm Dockerfile
	docker tag ${SRV_NAME}:latest ${NAME}:latest
	docker tag ${SRV_NAME}:latest ${NAME}:${TAG}
	docker push ${NAME}:latest
	docker push ${NAME}:${TAG}

pull:
	git pull origin ${BRANCH}

push:
	git push origin ${BRANCH}

tf:
	- terraform  -chdir=./terraform init --upgrade
	terraform -chdir=./terraform validate
	terraform -chdir=./terraform plan
	terraform -chdir=./terraform apply -auto-approve

del:
	terraform -chdir=./terraform destroy -auto-approve

dev:
	docker-compose up -d

down:
	docker-compose down
log:
	docker-compose logs -f

redis: helm
	helm upgrade --install redis-cluster bitnami/redis -f ./helm/redis/values.yaml --namespace redis --create-namespace

helm:
	helm repo add bitnami https://charts.bitnami.com/bitnami

secret:
	kubectl create secret docker-registry github-registry-secret \
		--docker-server=ghcr.io \
		--docker-username=${GITHUB_USERNAME} \
		--docker-password=${GITHUB_PERSONAL_ACCESS_TOKEN} \
		-o yaml > k8s/secret.yaml

d:
	kubectl delete -f ./k8s/job.yaml --ignore-not-found
	kubectl apply -f ./k8s
	# kubectl apply -f ./k8s/deployment.yaml

stats:
	kubectl delete -f ./k8s/stats.yaml --ignore-not-found
	kubectl apply -f ./k8s/stats.yaml

top:
	# kubectl top nodes
	kubectl get deployment coredns -n kube-system -o yaml > ./k8s/coredns.yaml
	