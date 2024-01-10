#!/bin/bash

show_invalid_usage() {
    echo "Needs at least two commands: $0 <command>";
    echo "";
    echo "Use --help to see available options";
}

conver () {
    t="/tmp/coverage.tmp"
    go test -coverprofile=$t ./http-adapter && go tool cover -html=$t && unlink $t
}

if [[ $# -lt 1 ]]; then
    show_invalid_usage
    exit -1;
fi;

case "$1" in 
    -h | --help)
        echo -e "\nCommand usage: $0 <command> [options]"
        echo ""
        echo -e "-c, --coverage    Test AND open coverage results in browser"
        echo -e "-h, --help        View this help prompt"
        echo -e "-t, --test        Run Unit Tests with coverage summary"
        echo -e "-b, --build       Build main.go"
        echo -e "-r, --run         Run the project"
        echo -e "-d, --docker      Run the project in docker"
    ;;
    -c | --coverage)
        cover    
    ;;
    -t | --test)
        go test -v -coverprofile="/tmp/coverage.tmp" --cover --tags "unit" ./...
    ;;
    -b | --build)
        go build -o bin/service main.go
    ;;
    -r | --run)
        go run main.go
    ;;
    -d | --docker)
        GOOS=linux GOARCH=amd64 go build -o ./build/go-test-linux-amd64 main.go && docker-compose up --build
    ;;
    *)
        show_invalid_usage
        exit -1;
    ;;
esac;


