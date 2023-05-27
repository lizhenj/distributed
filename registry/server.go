package registry

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
)

const (
	ServerPort = ":3000"
	ServicesURL = "http://localhost" + ServerPort + "/services"
)

type registry struct {
	registration []Registration
	mutex *sync.Mutex
}

func (r *registry) add(reg Registration) error{
	r.mutex.Lock()
	r.registration = append(r.registration,reg)
	r.mutex.Unlock()
	return nil
}

func (r *registry) remove(url string) error {
	for i := range r.registration {
		if reg.registration[i].ServiceURL == url{
			r.mutex.Lock()
			reg.registration = append(reg.registration[:i],reg.registration[i+1:]...)
			r.mutex.Unlock()
			return nil
		}
	}
	return fmt.Errorf("Service at URL %s not found",url)
}

var reg = registry{
	registration: make([]Registration,0),
	mutex: new(sync.Mutex),
}

type RegistryService struct {

}

func (s RegistryService) ServeHTTP(w http.ResponseWriter,r *http.Request){
	log.Println("Request received")
	switch r.Method {
	case http.MethodPost:
		dec := json.NewDecoder(r.Body)
		var r Registration
		err := dec.Decode(&r)
		if err != nil{
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Printf("Adding service: %v with URL: %s\n",
			r.ServiceName,r.ServiceURL)
		err = reg.add(r)
		if err != nil{
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	case http.MethodDelete:
		payload, err := ioutil.ReadAll(r.Body)
		if err != nil{
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		url := string(payload)
		log.Printf("Removing service at URL: %s",url)
		err = reg.remove(url)
		if err != nil{
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

