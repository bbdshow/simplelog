write_test:
	rm -rf ./testdata
	mkdir -p ./testdata
	go test  -v write.go write_test.go


clean:
	rm -rf ./testdata