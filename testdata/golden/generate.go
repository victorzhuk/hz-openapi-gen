//go:build ignore

package main

//go:generate go run gitlab.wildberries.ru/travel/travel-backend/adv/core/hz-openapi-gen/cmd/hz-openapi-gen -spec=testdata/minimal.yaml -out=. -module=example.com/service

func main() {}
