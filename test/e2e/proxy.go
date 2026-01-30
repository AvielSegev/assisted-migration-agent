package main

type Request struct {
	Headers      map[string]string
	RequestBody  []byte
	ResponseBody []byte
}
type Proxy struct{}
