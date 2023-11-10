operator="metric-collector"

export GO111MODULE=on
go mod vendor

go build -o ../build/_output/bin/$operator -mod=vendor ../cmd/main.go 
