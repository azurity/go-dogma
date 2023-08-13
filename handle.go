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

type Context struct {
	session any
}

type SessionManager interface {
	Load(writer http.ResponseWriter, request *http.Request) any
	Store(writer http.ResponseWriter, request *http.Request, session any)
}

type Method struct {
	Name   string `json:"name"`
	Method string `json:"method"`
}

type Server struct {
	router     *mux.Router
	sessionMan SessionManager
	desc       map[reflect.Type]Method
	logger     *log.Logger
}

func New(router *mux.Router, sessionMan SessionManager, desc map[reflect.Type]Method, logger *log.Logger) *Server {
	return &Server{
		router:     router,
		sessionMan: sessionMan,
		desc:       desc,
		logger:     logger,
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

type ResultBase struct {
	Message string `json:"message"`
}

func HandleRestFunc[T func(Context, PU, PC) (R, error), PU any, PC any, R any](s *Server, handle T) {
	typ := reflect.TypeOf(new(T))
	desc, ok := s.desc[typ]
	if !ok {
		log.Panicln(fmt.Sprintf("no such api for type: %s", typ.String()))
	}
	s.router.HandleFunc(fmt.Sprintf("/%s", desc.Name), func(writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)
		varsRaw, err := json.Marshal(vars)
		if err != nil {
			if s.logger != nil {
				s.logger.Println("[dogma]", err)
			}
			writer.WriteHeader(http.StatusBadRequest)
		}
		urlParam, err := typedParse[PU](varsRaw)
		if err != nil {
			if s.logger != nil {
				s.logger.Println("[dogma]", err)
			}
			writer.WriteHeader(http.StatusBadRequest)
		}
		commonParam := new(PC)
		if desc.Method == http.MethodPost {
			data, err := io.ReadAll(request.Body)
			if err != nil {
				if s.logger != nil {
					s.logger.Println("[dogma]", err)
				}
				writer.WriteHeader(http.StatusBadRequest)
			}
			commonParam, err = typedParse[PC](data)
			if err != nil {
				if s.logger != nil {
					s.logger.Println("[dogma]", err)
				}
				writer.WriteHeader(http.StatusBadRequest)
			}
		}
		ctx := Context{session: s.sessionMan.Load(writer, request)}
		ret, err := handle(ctx, *urlParam, *commonParam)
		s.sessionMan.Store(writer, request, ctx.session)
		result := struct {
			ResultBase
			R
		}{}
		if err != nil {
			result.Message = err.Error()
		} else {
			result.R = ret
		}
		raw, err := json.Marshal(ret)
		if err != nil {
			if s.logger != nil {
				s.logger.Println("[dogma]", err)
			}
			writer.WriteHeader(http.StatusBadRequest)
		}
		writer.Write(raw)
	}).Methods(desc.Method)
}
