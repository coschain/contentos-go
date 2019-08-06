COSD = github.com/coschain/contentos-go/cmd/cosd
WALLET = github.com/coschain/contentos-go/cmd/wallet-cli
MULTINODETESTER = github.com/coschain/contentos-go/cmd/multinodetester
DATABACKUP = github.com/coschain/contentos-go/cmd/databackup
NODEDETECTOR = github.com/coschain/contentos-go/cmd/nodedetector
PRESSURETEST = github.com/coschain/contentos-go/cmd/pressuretest

build_all:
	@echo "--> build all"
	@GO111MODULE=on go build -o ./bin/cosd $(COSD)
	@GO111MODULE=on go build -o ./bin/wallet-cli $(WALLET)
	@GO111MODULE=on go build -o ./bin/multinodetester $(MULTINODETESTER)
	@GO111MODULE=on go build -o ./bin/databackup $(DATABACKUP)
	@GO111MODULE=on go build -o ./bin/nodedetector $(NODEDETECTOR)
	@GO111MODULE=on go build -o ./bin/pressuretest $(PRESSURETEST)

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
