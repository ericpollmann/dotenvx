test:
	go test ./... -cover -count 1
	docker build -t dotenvx-decrypt .
	docker images | head -2
	docker run -e DOTENV_PRIVATE_KEY="2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d" dotenvx-decrypt
	docker run -e DOTENV_PRIVATE_KEY_PRODUCTION="7d797417f477635f8753c5325d5a68552ab7048f46c518be7f0ae3bc245d3ab8" dotenvx-decrypt

commit: test
	go fmt ./...
	git add -A
	git commit -m "$(MESSAGE)"
	git push -u origin main --force-with-lease
