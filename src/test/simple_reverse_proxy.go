package main
 
import (
	"log"
	"net/http"
	"net/http/httputil"
	"time"
)

func makeHandle(proxy http.Handler) func
	(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		dump, _ := httputil.DumpRequest(r, true)
		log.Println("Default handler:", r.Method, r.URL, string(dump))
		proxy.ServeHTTP(w, r)
	}
}

func main() {
	reverse_proxy := &httputil.ReverseProxy{
		func(r *http.Request) {},
		&http.Transport{Proxy: http.ProxyFromEnvironment,
		        DisableCompression: true},
		time.Duration(0)}
	http.HandleFunc("/", makeHandle(reverse_proxy))
	
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
	log.Println("Server started")
}
