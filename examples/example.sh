#!/usr/bin/env bash

(go run ./example.go &)
(go run ../. run -s ./example.json -x search &)

sleep 5

for i in `seq 1 10`;
do
	curl --output /dev/null --silent --location --request GET 'localhost:80/search/Istanbul'
done