//go:build ignore

package main

//go:generate go run github.com/victorzhuk/hz-openapi-gen -spec=testdata/minimal.yaml -out=. -module=example.com/service

func main() {}
