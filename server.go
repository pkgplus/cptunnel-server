package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rancher/remotedialer"
	"github.com/sirupsen/logrus"
)

var (
	port                   int
	timeout                int64
	bindAddr, domainSuffix string
	signKey                string
)

func init() {
	flag.IntVar(&port, "port", 80, "port")
	flag.StringVar(&bindAddr, "bind", "", "bind addr")
	flag.Int64Var(&timeout, "timeout", 15, "tunnel request timeout")
	flag.StringVar(&domainSuffix, "domain", "cmd.plus", "domain")
	flag.StringVar(&signKey, "key", "c8706bab7db59103a6bfd36e0c6b42e35d3f55d5", "sign key")
	flag.Parse()

	//logrus.SetLevel(logrus.DebugLevel)
	//remotedialer.PrintTunnelData = true
}

func main() {
	s := NewServer(port, bindAddr, timeout)
	s.Serve()
}

type Server struct {
	BindAddr string
	Port     int
	Timeout  time.Duration

	sync.Mutex
	server  *remotedialer.Server
	clients map[string]*http.Client

	agents sync.Map
}

func NewServer(port int, bindAddr string, timeout int64) *Server {
	return &Server{
		Port:     port,
		BindAddr: bindAddr,
		Timeout:  time.Duration(timeout) * time.Second,
		clients:  map[string]*http.Client{},
		agents:   sync.Map{},
	}
}

// Serve traffic
func (s *Server) Serve() {
	s.server = remotedialer.New(s.authorized, errorWriter)
	http.HandleFunc("/tunnel", s.tunnel)
	http.HandleFunc("/", s.dockerSocket)
	http.HandleFunc("/-/agents", s.onlineAgents)
	http.HandleFunc("/-/agent", s.onlineAgent)

	logrus.Infof("Data Plane Listening on %s:%d\n", s.BindAddr, s.Port)
	if err := http.ListenAndServe(fmt.Sprintf("%s:%d", s.BindAddr, s.Port), nil); err != nil {
		logrus.Fatal(err)
	}
}

func (s *Server) tunnel(w http.ResponseWriter, r *http.Request) {
	s.server.ServeHTTP(w, r)

	// offline
	username, _, ok := r.BasicAuth()
	if ok && username != "" {
		s.removeAgent(username)
	}
}

func (s *Server) dockerSocket(w http.ResponseWriter, r *http.Request) {
	clientKey := r.Host
	if strings.Contains(clientKey, ":") {
		clientKey = strings.SplitN(clientKey, ":", 2)[0]
	}
	clientKey = strings.TrimSuffix(clientKey, "."+domainSuffix)

	client := s.getClient(clientKey, s.Timeout)

	url := fmt.Sprintf("http://%s%s", "docker", r.RequestURI)
	p, _ := http.NewRequest(r.Method, url, r.Body)

	resp, err := client.Do(p)
	if err != nil {
		remotedialer.DefaultErrorWriter(w, r, 500, err)
		return
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (s *Server) onlineAgents(w http.ResponseWriter, r *http.Request) {
	agents := make([]*agent, 0)
	s.agents.Range(func(key, value interface{}) bool {
		agents = append(agents, value.(*agent))
		return true
	})
	jsonWriter(w, r, agents)
}

func (s *Server) onlineAgent(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("name")
	if username == "" {
		errorWriter(w, r, 400, errors.New("missing \"name\" query param"))
		return
	}

	agent, found := s.agents.Load(username)
	if !found {
		errorWriter(w, r, 400, errors.New("agent not online"))
		return
	}

	jsonWriter(w, r, agent)
}

func (s *Server) getClient(clientKey string, timeout time.Duration) *http.Client {
	s.Lock()
	defer s.Unlock()

	if s.clients == nil {
		s.clients = map[string]*http.Client{}
	}

	// unix docker socket
	var dialer remotedialer.Dialer
	dialer = func(proto, address string) (net.Conn, error) {
		return s.server.Dial(clientKey, 15*time.Second, "unix", "/var/run/docker.sock")
	}

	// http client
	client := &http.Client{
		Transport: &http.Transport{
			Dial: dialer,
		},
		Timeout: timeout,
	}

	s.clients[clientKey] = client
	return client
}

func (s *Server) authorized(req *http.Request) (id string, ok bool, err error) {
	username, password, ok := req.BasicAuth()
	if !ok {
		return "", false, nil
	}

	defer func() {
		if id == "" {
			ok = false
		}
	}()

	// TODO: check password
	h := hmac.New(sha256.New, []byte(signKey))
	io.WriteString(h, username)
	if fmt.Sprintf("%x", h.Sum(nil)) != password {
		return username, false, fmt.Errorf("password incorrect")
	}

	// online agent
	clientIP := req.Header.Get("X-Real-Ip")
	if clientIP == "" {
		clientIP = req.Header.Get("X-Forwarded-For")
	}
	err = s.addAgent(&agent{
		username,
		clientIP,
		time.Now(),
	})
	if err != nil {
		return username, false, err
	}

	return username, true, nil
}

func (s *Server) addAgent(agent *agent) (err error) {
	s.agents.Store(agent.UserName, agent)
	return nil
}

func (s *Server) removeAgent(username string) (err error) {
	s.agents.Delete(username)
	return nil
}

type agent struct {
	UserName string    `json:"username"`
	IP       string    `json:"ip"`
	Time     time.Time `json:"time"`
}

func errorWriter(rw http.ResponseWriter, req *http.Request, code int, err error) {
	rw.WriteHeader(code)

	bytes, _ := json.Marshal(map[string]string{
		"message": err.Error(),
	})

	rw.Write(bytes)
	rw.Header().Set("Content-Type", "application/json")
}

func jsonWriter(rw http.ResponseWriter, req *http.Request, v interface{}) {
	bytes, _ := json.Marshal(v)
	rw.Write(bytes)
	rw.Header().Set("Content-Type", "application/json")
}
