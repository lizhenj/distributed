package registry

import (
	"bytes"
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
	mutex *sync.RWMutex
}

func (r *registry) add(reg Registration) error{
	r.mutex.Lock()
	r.registration = append(r.registration,reg)
	r.mutex.Unlock()
	err := r.sendRequiredServices(reg)
	r.notify(patch{
		Added: []patchEntry{
			patchEntry{
				Name: reg.ServiceName,
				URL: reg.ServiceURL,
			},
		},
	})
	return err
}

func (r registry) notify(fullPatch patch) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	for _,reg := range r.registration{
		go func(reg Registration){
			for _,reqService := range reg.RequiredServices{
				p := patch{Added: []patchEntry{},Remove:[]patchEntry{}}
				sendUpdate := false
				for _,added := range fullPatch.Added {
					if added.Name == reqService{
						p.Added = append(p.Added,added)
						sendUpdate = true
					}
				}
				for _,remove := range fullPatch.Remove{
					if remove.Name == reqService{
						p.Remove = append(p.Remove,remove)
						sendUpdate = true
					}
				}
				if sendUpdate {
					err := r.sendPatch(p,reg.ServiceUpdateURL)
					if err != nil{
						log.Println(err)
						return
					}
				}
			}
		}(reg)
	}
}

func(r registry) sendRequiredServices(reg Registration) error{
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var p patch
	for _, serviceReg := range r.registration {
		for _,reqService := range reg.RequiredServices{
			if serviceReg.ServiceName == reqService{
				p.Added = append(p.Added,patchEntry{
					Name:serviceReg.ServiceName,
					URL:serviceReg.ServiceURL,
				})
			}
		}
	}
	err := r.sendPatch(p,reg.ServiceUpdateURL)
	if err != nil{
		return err
	}
	return nil
}

func (r registry) sendPatch(p patch,url string) error {
	d,err := json.Marshal(p)
	if err != nil{
		return err
	}
	_,err = http.Post(url,"application/json",bytes.NewBuffer(d))
	if err != nil{
		return err
	}
	return nil
}

func (r *registry) remove(url string) error {
	for i := range r.registration {
		if reg.registration[i].ServiceURL == url{
			r.notify(patch{
				Remove: []patchEntry{
					{
						Name:r.registration[i].ServiceName,
						URL: r.registration[i].ServiceURL,
					},
				},
			})
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
	mutex: new(sync.RWMutex),
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

