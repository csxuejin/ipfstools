mac:
	go build -o ipfstool *.go

linux:
	GOOS=linux GOARCH=amd64 go build -o ipfstool *.go
