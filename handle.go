package dogma

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"reflect"
)

type Method struct {
	Name   string `json:"name"`
	Method string `json:"method"`
}

type Server struct {
	router *mux.Router
	desc   map[reflect.Type]Method
}

func New(router *mux.Router, desc map[reflect.Type]Method) *Server {
	return &Server{
		router: router,
		desc:   desc,
	}
}

func typedParse[T any](data []byte) (*T, error) {
	ret := new(T)
	err := json.Unmarshal(data, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func HandleRestFunc[T func(PU, PC) (R, error), PU any, PC any, R any](s *Server, handle T) {
	typ := reflect.TypeOf(new(T))
	desc, ok := s.desc[typ]
	if !ok {
		log.Panicln(fmt.Sprintf("no such api for type: %s", typ.String()))
	}
	s.router.HandleFunc(fmt.Sprintf("/%s", desc.Name), func(writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)
		varsRaw, err := json.Marshal(vars)
		if err != nil {
			// TODO:
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		urlParam, err := typedParse[PU](varsRaw)
		if err != nil {
			// TODO:
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		data, err := io.ReadAll(request.Body)
		if err != nil {
			// TODO:
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		commonParam, err := typedParse[PC](data)
		if err != nil {
			// TODO:
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		ret, err := handle(*urlParam, *commonParam)
		if err != nil {
			// TODO:
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		raw, err := json.Marshal(ret)
		if err != nil {
			// TODO:
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		writer.Write(raw)
	}).Methods(desc.Method)
}
