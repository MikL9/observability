lint: betteralign golangci-lint goimports
	# TODO
	# echo "Запуск goimports"
	# find . -name "*.go" ! -path "*/.go*" ! -path "*/mocks/*" ! -path "*/mock_*.go" ! -path "*/generated/*" ! -path "./vendor/*" -exec go tool goimports -w -local credit-balance {} ;

goimports:
	echo "Запуск goimports"
	@find . -name "*.go" ! -path "*/.go*" ! -path "*/mocks/*" ! -path "*/mock_*.go" ! -path "*/generated/*" ! -path "./vendor/*" -exec go tool goimports -w -local credit-balance {} \;

betteralign:
	echo "Запуск betteralign"
	go tool betteralign -apply ./...

golangci-lint:
	echo "Запуск golangci-lint"
	go tool golangci-lint run --fix