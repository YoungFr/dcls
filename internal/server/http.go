package server

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func NewHTTPServer(addr string) *http.Server {
	s := newHTTPServer()
	r := mux.NewRouter()

	r.HandleFunc("/", s.handleWrite).Methods("POST")
	r.HandleFunc("/", s.handleRead).Methods("GET")

	return &http.Server{
		Addr: addr,
		// 一个 Handler 需要实现 ServeHTTP(ResponseWriter, *Request) 方法
		Handler: r,
	}
}

type HTTPServer struct {
	Log *Log
}

func newHTTPServer() *HTTPServer {
	return &HTTPServer{
		Log: NewLog(),
	}
}

type WriteReq struct {
	Record Record `json:"record"`
}

type WriteRsp struct {
	Offset uint64 `json:"offset"`
}

func (hs *HTTPServer) handleWrite(w http.ResponseWriter, r *http.Request) {
	// 1. 反序列化请求
	var req WriteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest) // 400
		return
	}
	// 2. 处理请求
	offset, err := hs.Log.Append(req.Record)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // 500
		return
	}
	rsp := WriteRsp{
		Offset: offset,
	}
	// 3. 序列化结果作为响应
	if err = json.NewEncoder(w).Encode(rsp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // 500
		return
	}
}

type ReadReq struct {
	Offset uint64 `json:"offset"`
}

type ReadRsp struct {
	Record Record `json:"record"`
}

func (hs *HTTPServer) handleRead(w http.ResponseWriter, r *http.Request) {
	// 1. 反序列化请求
	var req ReadReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest) // 400
		return
	}
	// 2. 处理请求
	record, err := hs.Log.Read(req.Offset)
	if err == errOffsetNotFound {
		http.Error(w, err.Error(), http.StatusNotFound) // 404
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // 500
		return
	}
	rsp := ReadRsp{
		Record: record,
	}
	// 3. 序列化结果作为响应
	if err := json.NewEncoder(w).Encode(rsp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // 500
		return
	}
}
