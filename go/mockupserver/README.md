# Mockup server

This is a companion server for testing the mobile implementation.

It can be run directly with go or using docker.

* Go: `go run . -ip 192.168.68.126 -issuetime 110 -verifytime 10`
* Docker: 
    1. Run this only once (or when code changes): `sudo docker build -t iden3mockupserver .`
    2. Run the server: `sudo docker run --rm -e OPTS="-ip 192.168.68.126 -issuetime 110 -verifytime 10" -p 1234:1234 iden3mockupserver`

The options of the program are:

| Flag        | Description                                     | Mandatory |
| ----------- | ----------------------------------------------- | :-------: |
| -ip         | IP of the machine where this software will run. |    YES    |
| -issuetime  | Time that takes to build a claim (in seconds).  |     NO    |
| -verifytime | Time that takes to verify a claim (in seconds). |     NO    |

