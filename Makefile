write_test:
	rm -rf ./testdata
	go test  -v write.go write_test.go

format_test:
	go test -v format.go format_test.go

simple_test:
	go test -count=1 -v simple_log.go format.go write.go simple_log_test.go

clean:
	rm -rf ./testdata