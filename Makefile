# build
.PHONY : build txstorm
build :
	go build -o build/benchopera ./cmd/benchopera

#test
.PHONY : test
test :
	go test ./...

#clean
.PHONY : clean
clean :
	rm ./build/benchopera
