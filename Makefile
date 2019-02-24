COSD = github.com/coschain/contentos-go/cmd/cosd
WALLET = github.com/coschain/contentos-go/cmd/wallet-cli

test:
	@echo "--> Running go test"
	@GO111MODULE=on go test -coverprofile=cc0.txt ./...
	@echo "--> Total code coverage"
	@GO111MODULE=on go run utils/totalcov/main.go . cc0.txt >coverage.txt

build:
	@echo "--> build all"
	@GO111MODULE=on go build -o ./bin/cosd $(COSD)
	@GO111MODULE=on go build -o ./bin/wallet-cli $(WALLET)

build_cosd:
	@echo "--> build cosd"
	@GO111MODULE=on go build -o ./bin/cosd $(COSD)

build_wallet:
	@echo "--> build wallet"
	@GO111MODULE=on go build -o ./bin/wallet-cli $(WALLET)

install:
	@echo "--> build all"
	@GO111MODULE=on go install $(COSD)
	@GO111MODULE=on go install $(WALLET)

install_cosd:
	@echo "--> install cosd"
	@GO111MODULE=on go install $(COSD)

install_wallet:
	@echo "--> build wallet"
	@GO111MODULE=on go install $(WALLET)
